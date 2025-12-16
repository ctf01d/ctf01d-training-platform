require "set"

class GamesController < ApplicationController
  require "securerandom"
  before_action :require_admin, except: %i[index show]
  before_action :set_game, only: %i[ show edit update destroy ]
  before_action :set_game_for_manage, only: %i[ manage_services add_service remove_service ]
  before_action :require_admin, only: %i[ export_ctf01d export_ctf01d_options ]

  # GET /games
  def index
    @ongoing_games  = Game.ongoing.order(ends_at: :asc)
    @upcoming_games = Game.upcoming.order(starts_at: :asc)
    @past_games     = Game.past.order(ends_at: :desc)
  end

  # GET /games/1
  def show
    if @game.final_results.exists?
      @final = true
      @results = @game.final_results.includes(:team).order(position: :asc, score: :desc)
    else
      @final = false
      @results = @game.results.includes(:team).order(score: :desc)
    end
    @game_teams = @game.game_teams.includes(:team).order(:order, :id)
    @game_team_lookup = @game_teams.index_by(&:team_id)
    @writeups = @game.writeups.includes(:team).order(created_at: :desc)
    if user_signed_in?
      # команды пользователя, которыми он может управлять
      my_team_ids = TeamMembership.where(user_id: current_user.id, status: TeamMembership::STATUS_APPROVED).pluck(:team_id)
      @my_manageable_teams = Team.where(id: my_team_ids).select { |t| can_manage_team?(t) }
    else
      @my_manageable_teams = []
    end
    if user_signed_in? && current_user.role == "admin"
      taken_ids = @game_teams.map(&:team_id)
      @available_teams = Team.where.not(id: taken_ids).order(:name)
      @new_game_team = GameTeam.new(game: @game, order: @game_teams.size + 1)
    else
      @available_teams = []
      @new_game_team = nil
    end
    @can_access = can_access_game?(@game)
  end

  # GET /games/:id/manage_services
  def manage_services
    @services = Service.order(:name)
    @attached_ids = @game.services.pluck(:id).to_set
  end

  # POST /games/:id/add_service
  def add_service
    service = Service.find(params.expect(:service_id))
    @game.services << service unless @game.services.exists?(service.id)
    redirect_to manage_services_game_path(@game), notice: "Сервис добавлен."
  end

  # DELETE /games/:id/remove_service
  def remove_service
    service = Service.find(params.expect(:service_id))
    @game.services.destroy(service)
    redirect_to manage_services_game_path(@game), notice: "Сервис удалён."
  end

  # GET /games/:id/export_ctf01d_options
  def export_ctf01d_options
    @game = Game.find(params.expect(:id))
    @form = {
      port: 8080,
      include_html: true,
      include_compose: true,
      flag_ttl_min: 1,
      basic_attack_cost: 1,
      defence_cost: 1.0,
      coffee_break_start: nil,
      coffee_break_end: nil
    }
  end

  # GET /games/:id/export_ctf01d
  # Генерация zip-архива для ctf01d (шаблон с примерами до настройки маппинга)
  def export_ctf01d
    game = Game.find(params.expect(:id))

    # Базовые значения (если не заданы даты — возьмём сейчас + 6 часов)
    starts = game.starts_at || Time.current
    ends   = game.ends_at   || (starts + 6.hours)

    game_payload = {
      id: game.name.to_s.downcase.gsub(/[^a-z0-9]+/, "").slice(0, 24).presence || "game#{game.id}",
      name: game.name,
      start_utc: starts.utc,
      end_utc: ends.utc,
      flag_ttl_min: 1,
      basic_attack_cost: 1,
      defence_cost: 1.0
    }

    # Параметры из формы (если заданы)
    port = params[:port].present? ? params[:port].to_i : 8080
    flag_ttl_min = params[:flag_ttl_min].present? ? params[:flag_ttl_min].to_i : 1
    basic_attack_cost = params[:basic_attack_cost].present? ? params[:basic_attack_cost].to_i : 1
    defence_cost = params[:defence_cost].present? ? params[:defence_cost].to_f : 1.0
    if params[:coffee_break_start].present? && params[:coffee_break_end].present?
      begin
        game_payload[:coffee_break_start_utc] = Time.zone.parse(params[:coffee_break_start]).utc
        game_payload[:coffee_break_end_utc] = Time.zone.parse(params[:coffee_break_end]).utc
      rescue
        # игнорируем некорректный ввод — сервис провалидирует позже
      end
    end

    game_payload[:flag_ttl_min] = flag_ttl_min
    game_payload[:basic_attack_cost] = basic_attack_cost
    game_payload[:defence_cost] = defence_cost

    scoreboard_payload = { port: port, htmlfolder: "./html", random: false }

    # Команды: если в игре заведён список — используем его с порядком и кастомными полями
    game_team_records = game.game_teams.includes(:team).order(:order, :id).to_a
    if game_team_records.empty? && game.results.exists?
      game_team_records = game.results.includes(:team).order(score: :desc).map.with_index do |res, idx|
        GameTeam.new(game: game, team: res.team, order: idx + 1)
      end
    end

    teams = []
    used_team_ids = Set.new

    if game_team_records.any?
      game_team_records.each_with_index do |gt, idx|
        n = idx + 1
        team_model = gt.respond_to?(:team) ? gt.team : nil
        base_overrides = if gt.respond_to?(:ctf01d_extra_hash)
          gt.ctf01d_extra_hash
        elsif gt.respond_to?(:ctf01d_overrides)
          gt.ctf01d_overrides
        else
          {}
        end
        overrides = base_overrides.to_h.transform_keys { |k| k.to_s.strip }
        overrides.delete_if { |_k, v| v.to_s.strip.empty? }

        ctf_id = if gt.respond_to?(:ctf01d_id)
          gt.ctf01d_id
        else
          nil
        end
        ctf_id = overrides.delete("ctf01d_id") if ctf_id.to_s.strip.empty?
        overrides.delete("ctf01d_id")

        if gt.respond_to?(:team_type) && gt.team_type.present?
          overrides["ctf01d_type"] ||= gt.team_type
        end

        active_flag = true
        if overrides.key?("ctf01d_active")
          active_flag = ActiveModel::Type::Boolean.new.cast(overrides.delete("ctf01d_active"))
        end

        team_id_base = ctf_id.to_s.downcase.gsub(/[^a-z0-9]+/, "")
        team_id_base = format("t%02d", n) if team_id_base.empty?
        team_id = team_id_base
        suffix = 2
        while used_team_ids.include?(team_id)
          team_id = "#{team_id_base}#{suffix}"
          suffix += 1
        end
        used_team_ids << team_id

        ip = gt.respond_to?(:ip_address) ? gt.ip_address.to_s.presence : nil
        ip ||= "10.0.#{n}.1"

        logo_rel = "./html/images/teams/#{team_id}.png"
        logo_url = team_model&.avatar_url
        fallback_logo = Rails.root.join("ctf01d", "data_sample", "html", "images", "teams", format("team%02d.png", [ n, 30 ].min)).to_s

        teams << {
          id: team_id,
          name: team_model&.name || "Team ##{n}",
          active: active_flag,
          ip_address: ip,
          logo_rel: logo_rel,
          logo_url: logo_url.presence,
          logo_src: (logo_url.present? ? nil : fallback_logo),
          ctf01d_extra: overrides
        }
      end
    else
      # Заглушки t01..t06 из примера
      6.times do |i|
        n = i + 1
        team_id = format("t%02d", n)
        teams << {
          id: team_id,
          name: "Team ##{n}",
          active: true,
          ip_address: "10.0.#{n}.1",
          logo_rel: format("./html/images/teams/team%02d.png", n),
          logo_src: Rails.root.join("ctf01d", "data_sample", "html", "images", "teams", format("team%02d.png", n)).to_s
        }
      end
    end

    # Чекеры: по одному на каждый сервис игры; если нет сервисов — один пример
    sample_checker_dir = Rails.root.join("ctf01d", "data_sample", "checker_example_service1").to_s
    services = game.services.order(:name).to_a
    used_ids = Set.new
    mk_id = ->(name, idx) do
      base = name.to_s.downcase.gsub(/[^a-z0-9]+/, "_").gsub(/\A_+|_+\z/, "")
      base = "service#{idx}" if base.blank?
      id = base
      n = 1
      while used_ids.include?(id)
        id = "#{base}_#{n}"
        n += 1
      end
      used_ids << id
      id
    end

    warnings = []
    tmp_sources = nil

    if services.any?
      Dir.mktmpdir("ctf01d_export_sources_") do |srcdir|
        tmp_sources = srcdir
        checkers = services.each_with_index.map do |svc, idx|
          cid = mk_id.call(svc.name, idx + 1)
          bundle_path = nil
          checker_from_bundle = false
          begin
            bundle_path = ensure_service_bundle_path!(svc)
            checker_from_bundle = bundle_has_dir?(bundle_path, "checker")
            warnings << "Сервис '#{svc.name}': в архиве нет checker/ — добавлен заглушечный checker.py" unless checker_from_bundle
          rescue Ctf01d::ExportError => e
            warnings << "Сервис '#{svc.name}': #{e.message} — добавлен заглушечный архив"
            bundle_path = build_placeholder_bundle_zip!(dir: srcdir, service_name: svc.name, service_id: cid)
            checker_from_bundle = true
          end

          {
            id: cid,
            name: svc.name,
            enabled: true,
            script_wait: 10,
            round_sleep: 30,
            script_rel: nil,
            bundle_path: bundle_path,
            checker_from_bundle: checker_from_bundle
          }
        end

        include_html = ActiveModel::Type::Boolean.new.cast(params[:include_html]).nil? ? true : ActiveModel::Type::Boolean.new.cast(params[:include_html])
        include_compose = ActiveModel::Type::Boolean.new.cast(params[:include_compose]).nil? ? true : ActiveModel::Type::Boolean.new.cast(params[:include_compose])

        result = Ctf01d::ExportZip.call(
          game: game_payload,
          scoreboard: scoreboard_payload,
          teams: teams,
          checkers: checkers,
          options: {
            prefix: "ctf01d_#{game_payload[:id]}",
            include_html: include_html,
            html_source_path: Rails.root.join("ctf01d", "data_sample", "html").to_s,
            include_compose: include_compose,
            compose_project: game_payload[:id],
            warnings: warnings
          }
        )

        cookies.signed[:ctf01d_export_warnings] = { value: warnings.join("\n"), expires: 5.minutes } if warnings.any?
        send_data result[:data], filename: result[:filename], type: "application/zip"
      end
      return
    else
      checkers = [
        { id: "example_service1", name: "Service1", enabled: true, script_wait: 5, round_sleep: 15,
          script_rel: "./checker.py", files: [ { src: File.join(sample_checker_dir, "checker.py"), rel: "checker.py" } ] }
      ]
    end

    include_html = ActiveModel::Type::Boolean.new.cast(params[:include_html]).nil? ? true : ActiveModel::Type::Boolean.new.cast(params[:include_html])
    include_compose = ActiveModel::Type::Boolean.new.cast(params[:include_compose]).nil? ? true : ActiveModel::Type::Boolean.new.cast(params[:include_compose])

    result = Ctf01d::ExportZip.call(
      game: game_payload,
      scoreboard: scoreboard_payload,
      teams: teams,
      checkers: checkers,
      options: {
        prefix: "ctf01d_#{game_payload[:id]}",
        include_html: include_html,
        html_source_path: Rails.root.join("ctf01d", "data_sample", "html").to_s,
        include_compose: include_compose,
        compose_project: game_payload[:id],
        warnings: warnings
      }
    )

    send_data result[:data], filename: result[:filename], type: "application/zip"
  rescue Ctf01d::ExportError => e
    Rails.logger.error("[export_ctf01d] #{e.class}: #{e.message}\n#{e.backtrace&.join("\n")}")
    redirect_to game_path(game), alert: "Не удалось собрать архив: #{e.message}"
  rescue => e
    Rails.logger.error("[export_ctf01d] #{e.class}: #{e.message}\n#{e.backtrace&.join("\n")}")
    redirect_to game_path(game), alert: "Ошибка экспорта архива."
  end

  # GET /games/new
  def new
    @game = Game.new
  end

  # GET /games/1/edit
  def edit
  end

  # POST /games
  def create
    attrs = game_params
    upload = attrs.delete(:avatar_upload)
    @game = Game.new(attrs)
    apply_avatar(@game, upload, placeholder_dir: "game-logos")

    if @game.save
      redirect_to @game, notice: "Игра создана."
    else
      render :new, status: :unprocessable_content
    end
  end

  # PATCH/PUT /games/1
  def update
    attrs = game_params
    upload = attrs.delete(:avatar_upload)
    @game.assign_attributes(attrs)
    apply_avatar(@game, upload, placeholder_dir: "game-logos")

    if @game.save
      redirect_to @game, notice: "Игра обновлена.", status: :see_other
    else
      render :edit, status: :unprocessable_content
    end
  end

  # DELETE /games/1
  def destroy
    @game.destroy!
    redirect_to games_path, notice: "Игра удалена.", status: :see_other
  end

  # POST /games/:id/finalize
  def finalize
    game = Game.find(params.expect(:id))
    ActiveRecord::Base.transaction do
      # Prevent double-finalization
      if game.final_results.exists?
        game.update!(finalized: true, finalized_at: Time.current) unless game.finalized
      else
        rows = game.results.includes(:team).order(score: :desc).to_a
        rows.each_with_index do |r, idx|
          FinalResult.create!(game_id: game.id, team_id: r.team_id, score: r.score.to_i, position: idx + 1)
        end
        game.update!(finalized: true, finalized_at: Time.current)
      end
    end
    redirect_to game, notice: "Итоги зафиксированы."
  rescue => e
    redirect_to game, alert: "Не удалось зафиксировать итоги."
  end

  # DELETE /games/:id/unfinalize
  def unfinalize
    game = Game.find(params.expect(:id))
    ActiveRecord::Base.transaction do
      game.final_results.delete_all
      game.update!(finalized: false, finalized_at: nil)
    end
    redirect_to game, notice: "Финализация снята."
  rescue => e
    redirect_to game, alert: "Не удалось снять финализацию."
  end

  private
    # Use callbacks to share common setup or constraints between actions.
    def set_game
      @game = Game.find(params.expect(:id))
    end

    def set_game_for_manage
      require_admin
      @game = Game.find(params.expect(:id))
    end

  # Only allow a list of trusted parameters through.
  def game_params
      params.expect(game: [ :name, :organizer, :starts_at, :ends_at, :avatar_url, :site_url, :ctftime_url,
                            :registration_opens_at, :registration_closes_at,
                            :scoreboard_opens_at, :scoreboard_closes_at,
                            :vpn_url, :vpn_config_url, :access_secret, :access_instructions, :avatar_upload,
                            { service_ids: [] } ])
    end

    def apply_avatar(record, upload, placeholder_dir:)
      if upload.respond_to?(:original_filename)
        path = save_avatar_file(upload, folder: placeholder_dir.include?("game") ? "games" : "teams")
        record.avatar_url = path if path
      elsif record.avatar_url.blank?
        placeholder = pick_placeholder(record.name, base_dir: placeholder_dir)
        record.avatar_url = placeholder if placeholder
      end
    end

    def save_avatar_file(upload, folder:)
      ext = File.extname(upload.original_filename.to_s)
      ext = ".png" if ext.blank?
      fname = "#{Time.now.utc.strftime('%Y%m%d')}-#{SecureRandom.hex(6)}#{ext}"
      dir = Rails.root.join("public", "uploads", folder)
      FileUtils.mkdir_p(dir)
      File.open(dir.join(fname), "wb") { |f| f.write(upload.read) }
      "/uploads/#{folder}/#{fname}"
    end

    def pick_placeholder(name, base_dir:)
      slug = name.to_s.parameterize
      return nil if slug.blank?
      public_dir = Rails.root.join("public", "img", base_dir)
      source_dir = Rails.root.join("img", base_dir)
      if Dir.exist?(source_dir) && !Dir.exist?(public_dir)
        FileUtils.mkdir_p(public_dir)
        FileUtils.cp_r("#{source_dir}/.", public_dir)
      end
      candidates = Dir.glob(public_dir.join("#{slug}.*"))
      candidates = Dir.glob(public_dir.join("#{slug.tr('-', '_')}.*")) if candidates.empty?
      candidates.first&.sub(Rails.root.join("public").to_s, "")
    end

    def ensure_service_bundle_path!(service)
      rel = service.service_local_path.to_s
      abs = rel.present? ? Rails.root.join(rel).to_s : nil
      return abs if abs && File.file?(abs)

      if service.service_archive_url.present?
        ServiceArchives.redownload(service: service, kind: :service)
        rel = service.service_local_path.to_s
        abs = rel.present? ? Rails.root.join(rel).to_s : nil
        return abs if abs && File.file?(abs)
      end

      raise Ctf01d::ExportError, "у сервиса '#{service.name}' нет локального архива и отсутствует Archive URL"
    rescue ServiceArchives::Error => e
      raise Ctf01d::ExportError, "не удалось подготовить архив для сервиса '#{service.name}': #{e.message}"
    end

    def bundle_has_dir?(zip_path, dir_name)
      require "zip"
      want = %r{(^|/)#{Regexp.escape(dir_name)}/}
      Zip::File.open(zip_path) do |zip|
        zip.each do |e|
          n = e.name.to_s
          next if n.blank?
          return true if n =~ want
        end
      end
      false
    rescue Zip::Error
      false
    end

    def build_placeholder_bundle_zip!(dir:, service_name:, service_id:)
      require "zip"
      safe = service_id.to_s.downcase.gsub(/[^a-z0-9_]+/, "_").gsub(/\A_+|_+\z/, "")
      safe = "service" if safe.blank?
      path = File.join(dir, "#{safe}.zip")
      Zip::File.open(path, create: true) do |zip|
        zip.get_output_stream("service/README.md") { |f| f.write("placeholder for #{service_name}\n") }
        zip.get_output_stream("checker/checker.py") { |f| f.write("#!/usr/bin/env python3\nprint('placeholder checker')\n") }
      end
      path
    end
end
