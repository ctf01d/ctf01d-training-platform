class TeamMembershipsController < ApplicationController
  before_action :require_admin, except: %i[index show approve reject accept decline set_role destroy]
  before_action :set_team_membership, only: %i[ show edit update destroy approve reject accept decline set_role ]

  # GET /team_memberships
  def index
    @team_memberships = TeamMembership.all
  end

  # GET /team_memberships/1
  def show
  end

  # GET /team_memberships/new
  def new
    @team_membership = TeamMembership.new(
      team_id: params[:team_id],
      role: TeamMembership::ROLE_PLAYER,
      status: TeamMembership::STATUS_APPROVED
    )
  end

  # GET /team_memberships/1/edit
  def edit
  end

  # POST /team_memberships
  def create
    @team_membership = TeamMembership.new(team_membership_params)

    if @team_membership.save
      redirect_to @team_membership, notice: "Team membership was successfully created."
    else
      render :new, status: :unprocessable_content
    end
  end

  # PATCH/PUT /team_memberships/1
  def update
    if @team_membership.update(team_membership_params)
      redirect_to @team_membership, notice: "Team membership was successfully updated.", status: :see_other
    else
      render :edit, status: :unprocessable_content
    end
  end

  # DELETE /team_memberships/1
  def destroy
    team = @team_membership.team
    # Разрешаем удалять: админу, менеджерам команды (owner/captain/vice_captain) и самому пользователю (выйти из команды)
    unless user_signed_in? && (current_user.role == "admin" || can_manage_team?(team) || current_user.id == @team_membership.user_id)
      return redirect_to team, alert: "Недостаточно прав"
    end

    # Запрет удалять владельца не-админам
    if @team_membership.role == TeamMembership::ROLE_OWNER && !(user_signed_in? && current_user.role == "admin")
      return redirect_to team, alert: "Нельзя удалить владельца команды"
    end

    # Если удаляем капитана — очищаем привязку у команды
    if team.captain_id == @team_membership.user_id
      team.update(captain_id: nil)
    end

    from_role = @team_membership.role
    from_status = @team_membership.status
    user = @team_membership.user
    actor = user_signed_in? ? current_user : nil
    action = (actor && actor.id == user.id) ? "left" : "removed"
    @team_membership.destroy!
    TeamMembershipEvent.create!(team: team, user: user, actor: actor, action: action, from_role: from_role, from_status: from_status)
    redirect_to team, notice: "Участник удален.", status: :see_other
  end

  # POST /team_memberships/:id/approve
  def approve
    unless can_manage_team?(@team_membership.team)
      return redirect_to @team_membership.team, alert: "Недостаточно прав"
    end
    if user_signed_in? && current_user.id == @team_membership.user_id
      return redirect_to @team_membership.team, alert: "Нельзя одобрить свою заявку. Используйте «Принять»."
    end
    from = @team_membership.status
    if @team_membership.update(status: TeamMembership::STATUS_APPROVED)
      TeamMembershipEvent.create!(team: @team_membership.team, user: @team_membership.user, actor: current_user, action: "approved", from_status: from, to_status: "approved")
      redirect_to @team_membership.team, notice: "Заявка одобрена."
    else
      redirect_to @team_membership.team, alert: "Не удалось одобрить заявку."
    end
  end

  # POST /team_memberships/:id/reject
  def reject
    unless can_manage_team?(@team_membership.team)
      return redirect_to @team_membership.team, alert: "Недостаточно прав"
    end
    if user_signed_in? && current_user.id == @team_membership.user_id
      return redirect_to @team_membership.team, alert: "Нельзя отклонить свою заявку. Используйте «Отклонить» внизу."
    end
    from = @team_membership.status
    if @team_membership.update(status: TeamMembership::STATUS_REJECTED)
      TeamMembershipEvent.create!(team: @team_membership.team, user: @team_membership.user, actor: current_user, action: "rejected", from_status: from, to_status: "rejected")
      redirect_to @team_membership.team, notice: "Заявка отклонена."
    else
      redirect_to @team_membership.team, alert: "Не удалось отклонить заявку."
    end
  end

  # POST /team_memberships/:id/accept
  def accept
    @team_membership = TeamMembership.find(params.expect(:id))
    team = @team_membership.team
    # Принятие выполняет менеджмент команды через "Одобрить"
    redirect_to team, alert: "Принятие заявки выполняет менеджмент команды."
  end

  # POST /team_memberships/:id/decline
  def decline
    @team_membership = TeamMembership.find(params.expect(:id))
    unless user_signed_in? && current_user.id == @team_membership.user_id
      return redirect_to @team_membership.team, alert: "Недостаточно прав"
    end
    from = @team_membership.status
    if @team_membership.update(status: TeamMembership::STATUS_REJECTED)
      TeamMembershipEvent.create!(team: @team_membership.team, user: @team_membership.user, actor: current_user, action: "declined", from_status: from, to_status: "rejected")
      redirect_to @team_membership.team, notice: "Вы отклонили приглашение."
    else
      redirect_to @team_membership.team, alert: "Не удалось отклонить приглашение."
    end
  end

  # POST /team_memberships/:id/set_role
  def set_role
    @team_membership = TeamMembership.find(params.expect(:id))
    team = @team_membership.team
    unless can_manage_team?(team)
      return redirect_to team, alert: "Недостаточно прав"
    end
    new_role = params[:role].to_s
    unless TeamMembership::ROLES.include?(new_role)
      return redirect_to team, alert: "Недопустимая роль"
    end

    # Глобальное ограничение: пользователь может быть капитаном только в одной команде
    if new_role == TeamMembership::ROLE_CAPTAIN
      if Team.where(captain_id: @team_membership.user_id).where.not(id: team.id).exists?
        return redirect_to team, alert: "Этот пользователь уже является капитаном другой команды."
      end
    end

    ActiveRecord::Base.transaction do
      if new_role == TeamMembership::ROLE_CAPTAIN
        # Сбрасываем предыдущего капитана
        if team.captain_id.present? && team.captain_id != @team_membership.user_id
          prev = TeamMembership.find_by(team_id: team.id, user_id: team.captain_id)
          prev.update!(role: TeamMembership::ROLE_PLAYER) if prev&.role == TeamMembership::ROLE_CAPTAIN
        end
        team.update!(captain_id: @team_membership.user_id)
      else
        # Если снимаем капитана, чистим связь
        if team.captain_id == @team_membership.user_id
          team.update!(captain_id: nil)
        end
      end

      from_role = @team_membership.role
      @team_membership.update!(role: new_role, status: TeamMembership::STATUS_APPROVED)
      TeamMembershipEvent.create!(team: team, user: @team_membership.user, actor: current_user, action: "role_changed", from_role: from_role, to_role: new_role)
    end

    redirect_to team, notice: "Роль обновлена."
  rescue => e
    redirect_to team, alert: "Не удалось обновить роль."
  end

  private
    # Use callbacks to share common setup or constraints between actions.
    def set_team_membership
      @team_membership = TeamMembership.find(params.expect(:id))
    end

    # Only allow a list of trusted parameters through.
    def team_membership_params
      params.expect(team_membership: [ :team_id, :user_id, :role, :status ])
    end
end
