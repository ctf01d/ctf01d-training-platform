class ApplicationController < ActionController::Base
  # Only allow modern browsers supporting webp images, web push, badges, import maps, CSS nesting, and CSS :has.
  allow_browser versions: :modern

  before_action :consume_export_warnings_cookie

  helper_method :current_user, :user_signed_in?, :can_manage_team?
  helper_method :can_access_game?

  private
  def consume_export_warnings_cookie
    return unless request.format.html?
    msg = cookies.signed[:ctf01d_export_warnings].to_s
    return if msg.blank?
    cookies.delete(:ctf01d_export_warnings)
    flash.now[:warning] = msg
  end

  def current_user
    return @current_user if defined?(@current_user)
    @current_user = User.find_by(id: session[:user_id])
  end

  def user_signed_in?
    current_user.present?
  end

  def require_login
    redirect_to new_session_path, alert: "Требуется авторизация" unless user_signed_in?
  end

  def require_admin
    unless user_signed_in? && current_user.role == "admin"
      redirect_to root_path, alert: "Недостаточно прав"
    end
  end

  def can_manage_team?(team)
    return true if user_signed_in? && current_user.role == "admin"
    return false unless user_signed_in?
    membership = TeamMembership.find_by(team_id: team.id, user_id: current_user.id, status: TeamMembership::STATUS_APPROVED)
    return false unless membership
    TeamMembership.manager_roles.include?(membership.role)
  end

  # Доступ к сетям/инфраструктуре игры
  def can_access_game?(game)
    return true if user_signed_in? && current_user.role == "admin"
    return false unless user_signed_in?
    team_ids = game.results.select(:team_id)
    return false if team_ids.blank?
    TeamMembership.where(team_id: team_ids, user_id: current_user.id, status: TeamMembership::STATUS_APPROVED).exists?
  end
end
