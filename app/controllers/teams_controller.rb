class TeamsController < ApplicationController
  require "securerandom"
  require "marcel"
  before_action :require_admin, except: %i[index show join_request]
  before_action :set_team, only: %i[ show edit update destroy join_request ]
  AvatarUploadError = Class.new(StandardError)

  # GET /teams
  def index
    @teams = Team.all
  end

  # GET /teams/1
  def show
  end

  # GET /teams/new
  def new
    @team = Team.new
  end

  # GET /teams/1/edit
  def edit
  end

  # POST /teams
  def create
    attrs = team_params
    upload = attrs.delete(:avatar_upload)
    @team = Team.new(attrs)
    begin
      apply_avatar(@team, upload, placeholder_dir: "team-logos")
    rescue AvatarUploadError => e
      @team.errors.add(:avatar_url, e.message)
      return render :new, status: :unprocessable_content
    end

    if @team.save
      if user_signed_in?
        # Назначаем создателя команды владельцем и (если возможно) капитаном
        if !@team.captain_id.present? && !Team.exists?(captain_id: current_user.id)
          @team.update(captain_id: current_user.id)
        end
        TeamMembership.find_or_create_by!(team_id: @team.id, user_id: current_user.id) do |m|
          m.role = TeamMembership::ROLE_OWNER
          m.status = TeamMembership::STATUS_APPROVED
        end
      end
      redirect_to @team, notice: "Команда создана."
    else
      render :new, status: :unprocessable_content
    end
  end

  # PATCH/PUT /teams/1
  def update
    attrs = team_params
    upload = attrs.delete(:avatar_upload)
    @team.assign_attributes(attrs)
    begin
      apply_avatar(@team, upload, placeholder_dir: "team-logos")
    rescue AvatarUploadError => e
      @team.errors.add(:avatar_url, e.message)
      return render :edit, status: :unprocessable_content
    end

    if @team.save
      if @team.saved_change_to_captain_id?
        old_id, new_id = @team.saved_change_to_captain_id

        if old_id.present?
          prev = TeamMembership.find_by(team_id: @team.id, user_id: old_id)
          if prev && prev.role == TeamMembership::ROLE_CAPTAIN
            # если бывший капитан был владельцем, оставляем owner, иначе делаем player
            prev_role_was = prev.role
            prev.update(role: TeamMembership::ROLE_PLAYER)
            TeamMembershipEvent.create!(team: @team, user: prev.user, actor: current_user, action: "role_changed", from_role: prev_role_was, to_role: TeamMembership::ROLE_PLAYER)
          end
        end

        if new_id.present?
          curr = TeamMembership.find_or_initialize_by(team_id: @team.id, user_id: new_id)
          from_role = curr.role
          curr.status = TeamMembership::STATUS_APPROVED
          curr.role = TeamMembership::ROLE_CAPTAIN
          curr.save!
          TeamMembershipEvent.create!(team: @team, user: curr.user, actor: current_user, action: "role_changed", from_role: from_role, to_role: TeamMembership::ROLE_CAPTAIN)
        end
      end
      redirect_to @team, notice: "Команда обновлена.", status: :see_other
    else
      render :edit, status: :unprocessable_content
    end
  end

  # DELETE /teams/1
  def destroy
    @team.destroy!
    redirect_to teams_path, notice: "Команда удалена.", status: :see_other
  end

  # POST /teams/:id/join_request
  def join_request
    return redirect_to new_session_path, alert: "Требуется авторизация" unless user_signed_in?

    membership = TeamMembership.find_by(team_id: @team.id, user_id: current_user.id)
    if membership&.status == TeamMembership::STATUS_APPROVED
      redirect_to @team, notice: "Вы уже участник команды."
    elsif membership&.status == TeamMembership::STATUS_PENDING
      redirect_to @team, notice: "Заявка уже подана."
    else
      membership ||= TeamMembership.new(team_id: @team.id, user_id: current_user.id)
      membership.role = TeamMembership::ROLE_PLAYER
      membership.status = TeamMembership::STATUS_PENDING
      if membership.save
        TeamMembershipEvent.create!(team: @team, user: current_user, actor: current_user, action: "join_requested", to_status: "pending")
        redirect_to @team, notice: "Заявка отправлена."
      else
        redirect_to @team, alert: "Не удалось отправить заявку."
      end
    end
  end

  # POST /teams/:id/invite
  def invite
    unless can_manage_team?(@team)
      return redirect_to @team, alert: "Недостаточно прав"
    end

    login = params[:user_name].to_s.strip
    user = User.find_by(user_name: login)
    return redirect_to @team, alert: "Пользователь не найден" unless user

    membership = TeamMembership.find_or_initialize_by(team_id: @team.id, user_id: user.id)
    if membership.status == TeamMembership::STATUS_APPROVED
      redirect_to @team, notice: "Пользователь уже в команде."
    else
      membership.role ||= TeamMembership::ROLE_PLAYER
      membership.status = TeamMembership::STATUS_PENDING
      if membership.save
        TeamMembershipEvent.create!(team: @team, user: user, actor: current_user, action: "invited", to_status: "pending")
        redirect_to @team, notice: "Приглашение отправлено."
      else
        redirect_to @team, alert: "Не удалось отправить приглашение."
      end
    end
  end

  private
    # Use callbacks to share common setup or constraints between actions.
    def set_team
      @team = Team.find(params.expect(:id))
    end

  # Only allow a list of trusted parameters through.
  def team_params
      params.expect(team: [ :name, :university_id, :description, :website, :avatar_url, :avatar_upload, :captain_id ])
    end

    def apply_avatar(record, upload, placeholder_dir:)
      if upload.respond_to?(:original_filename)
        path = save_avatar_file(upload, folder: "teams")
        record.avatar_url = path if path
      elsif record.avatar_url.blank?
        placeholder = pick_placeholder(record.name, base_dir: placeholder_dir)
        record.avatar_url = placeholder if placeholder
      end
    end

    def save_avatar_file(upload, folder:)
      max_bytes = 2.megabytes
      size = upload.respond_to?(:size) ? upload.size.to_i : nil
      raise AvatarUploadError, "слишком большой файл (макс #{max_bytes / 1024 / 1024} МБ)" if size && size > max_bytes

      detected = Marcel::MimeType.for(upload, name: upload.original_filename.to_s, declared_type: upload.content_type)
      ext = {
        "image/png" => ".png",
        "image/jpeg" => ".jpg",
        "image/webp" => ".webp",
        "image/gif" => ".gif"
      }[detected.to_s]
      raise AvatarUploadError, "поддерживаются только PNG/JPEG/WebP/GIF" if ext.blank?

      fname = "#{Time.now.utc.strftime('%Y%m%d')}-#{SecureRandom.hex(6)}#{ext}"
      dir = Rails.root.join("public", "uploads", folder)
      FileUtils.mkdir_p(dir)
      written = 0
      File.open(dir.join(fname), "wb") do |f|
        upload.rewind if upload.respond_to?(:rewind)
        while (chunk = upload.read(16 * 1024))
          break if chunk.empty?
          written += chunk.bytesize
          raise AvatarUploadError, "слишком большой файл (макс #{max_bytes / 1024 / 1024} МБ)" if written > max_bytes
          f.write(chunk)
        end
      end
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
end
