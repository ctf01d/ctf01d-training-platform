class GamesController < ApplicationController
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
    @writeups = @game.writeups.includes(:team).order(created_at: :desc)
    if user_signed_in?
      # команды пользователя, которыми он может управлять
      my_team_ids = TeamMembership.where(user_id: current_user.id, status: TeamMembership::STATUS_APPROVED).pluck(:team_id)
      @my_manageable_teams = Team.where(id: my_team_ids).select { |t| can_manage_team?(t) }
    else
      @my_manageable_teams = []
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
      coffee_break_end: nil,
      ip_pattern: "10.0.*.1"
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
    ip_pattern = params[:ip_pattern].present? ? params[:ip_pattern].to_s : "10.0.*.1"

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

    # Команды: если нет привязанных — создадим 6 заглушек из примера
    teams_scope = game.teams.presence || []
    teams = []
    if teams_scope.any?
      teams_scope.each_with_index do |t, idx|
        n = (idx + 1)
        team_id = format("t%02d", n)
        logo_rel = "./html/images/teams/#{team_id}.png"
        logo_url = t.respond_to?(:avatar_url) ? t.avatar_url.to_s : nil
        ip = t.respond_to?(:ip_address) ? t.ip_address.to_s.presence : nil
        ip ||= ip_pattern.to_s.gsub("{n}", n.to_s).gsub("*", n.to_s)
        fallback_logo = Rails.root.join("ctf01d", "data_sample", "html", "images", "teams", format("team%02d.png", [ n, 30 ].min)).to_s
        teams << {
          id: team_id,
          name: t.name,
          active: true,
          ip_address: ip,
          logo_rel: logo_rel,
          logo_url: logo_url.presence,
          logo_src: (logo_url.present? ? nil : fallback_logo)
        }
      end
    else
      # Заглушки t01..t06 из примера
      6.times do |i|
        n = i + 1
        teams << {
          id: format("t%02d", n),
          name: "Team ##{n}",
          active: true,
          ip_address: ip_pattern.to_s.gsub("{n}", n.to_s).gsub("*", n.to_s),
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

    if services.any?
      checkers = services.each_with_index.map do |svc, idx|
        cid = mk_id.call(svc.name, idx+1)
        {
          id: cid,
          name: svc.name,
          enabled: true,
          script_wait: 5,
          round_sleep: 15,
          script_rel: "./checker.py",
          files: [ { src: File.join(sample_checker_dir, "checker.py"), rel: "checker.py" } ]
        }
      end
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
        compose_project: game_payload[:id]
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
    @game = Game.new(game_params)

    if @game.save
      redirect_to @game, notice: "Игра создана."
    else
      render :new, status: :unprocessable_content
    end
  end

  # PATCH/PUT /games/1
  def update
    if @game.update(game_params)
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
                            :vpn_url, :vpn_config_url, :access_secret, :access_instructions,
                            { service_ids: [] } ])
    end
end
