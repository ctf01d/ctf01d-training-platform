class ApplicationController < ActionController::Base
  # Only allow modern browsers supporting webp images, web push, badges, import maps, CSS nesting, and CSS :has.
  allow_browser versions: :modern

  helper_method :current_user, :user_signed_in?, :can_manage_team?

  private

  def current_user
    return @current_user if defined?(@current_user)
    @current_user = User.find_by(id: session[:user_id])
  end

  def user_signed_in?
    current_user.present?
  end

  def require_login
    redirect_to new_session_path, alert: 'Требуется авторизация' unless user_signed_in?
  end

  def require_admin
    unless user_signed_in? && current_user.role == 'admin'
      redirect_to root_path, alert: 'Недостаточно прав'
    end
  end

  def can_manage_team?(team)
    return true if user_signed_in? && current_user.role == 'admin'
    return false unless user_signed_in?
    membership = TeamMembership.find_by(team_id: team.id, user_id: current_user.id, status: 'approved')
    return false unless membership
    %w[owner captain vice_captain].include?(membership.role)
  end
end
