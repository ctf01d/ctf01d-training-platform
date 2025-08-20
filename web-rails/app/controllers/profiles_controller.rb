class ProfilesController < ApplicationController
  before_action :require_login

  def show
    @user = current_user
    @memberships = @user.team_memberships.includes(:team).order(created_at: :desc)
  end

  def edit
    @user = current_user
  end

  def update
    @user = current_user
    if @user.update(profile_params)
      redirect_to profile_path, notice: 'Профиль обновлен.'
    else
      render :edit, status: :unprocessable_entity
    end
  end

  private

  def profile_params
    params.expect(user: [:display_name, :avatar_url, :password, :password_confirmation])
  end
end
