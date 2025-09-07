class ProfilesController < ApplicationController
  before_action :require_login

  def show
    @user = current_user
    @memberships = @user.team_memberships.includes(:team).order(created_at: :desc)

    # Фильтры по событиям: команда и тип действия
    team_filter = params[:team_id].presence
    action_filter = params[:action_type].presence

    events_scope = TeamMembershipEvent.where(user_id: @user.id)
    events_scope = events_scope.where(team_id: team_filter) if team_filter
    events_scope = events_scope.where(action: action_filter) if action_filter

    # Простая пагинация без гемов
    @per_page = (params[:per].presence || 20).to_i.clamp(1, 100)
    @page = (params[:page].presence || 1).to_i
    @total_events = events_scope.count
    @total_pages = (@total_events.to_f / @per_page).ceil
    @page = 1 if @page < 1
    @page = @total_pages if @total_pages > 0 && @page > @total_pages

    @events = events_scope.includes(:team, :actor).order(created_at: :desc)
                          .offset((@page - 1) * @per_page).limit(@per_page)
  end

  def edit
    @user = current_user
  end

  def update
    @user = current_user
    if @user.update(profile_params)
      redirect_to profile_path, notice: "Профиль обновлен."
    else
      render :edit, status: :unprocessable_content
    end
  end

  private

  def profile_params
    params.expect(user: [ :display_name, :avatar_url, :password, :password_confirmation ])
  end
end
