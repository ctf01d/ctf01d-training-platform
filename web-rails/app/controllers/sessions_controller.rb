class SessionsController < ApplicationController
  def new; end

  def create
    user = User.find_by(user_name: params[:user_name])
    if user&.authenticate(params[:password])
      session[:user_id] = user.id
      redirect_to root_path, notice: 'Успешный вход'
    else
      flash.now[:alert] = 'Неверный логин или пароль'
      render :new, status: :unprocessable_entity
    end
  end

  def destroy
    reset_session
    redirect_to root_path, notice: 'Вы вышли'
  end
end

