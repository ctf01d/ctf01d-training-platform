class GameTeamsController < ApplicationController
  before_action :require_admin
  before_action :set_game_team, only: %i[update destroy]

  def create
    @game_team = GameTeam.new(game_team_params)
    @game_team.order = GameTeam.where(game_id: @game_team.game_id).count + 1 if @game_team.order.blank?

    if @game_team.save
      redirect_to game_path(@game_team.game_id), notice: "Команда добавлена в игру."
    else
      game_id = @game_team.game_id || game_team_params[:game_id]
      redirect_to game_path(game_id), alert: @game_team.errors.full_messages.to_sentence
    end
  end

  def update
    if @game_team.update(game_team_params)
      redirect_to game_path(@game_team.game_id), notice: "Данные команды обновлены.", status: :see_other
    else
      redirect_to game_path(@game_team.game_id), alert: @game_team.errors.full_messages.to_sentence
    end
  end

  def destroy
    game_id = @game_team.game_id
    @game_team.destroy!
    redirect_to game_path(game_id), notice: "Команда убрана из игры.", status: :see_other
  end

  private

  def set_game_team
    @game_team = GameTeam.find(params.expect(:id))
  end

  def game_team_params
    params.expect(game_team: [ :game_id, :team_id, :ip_address, :order, :ctf01d_id, :team_type, :ctf01d_overrides_text ])
  end
end
