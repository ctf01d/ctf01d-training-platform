class WriteupsController < ApplicationController
  before_action :require_login
  before_action :set_writeup, only: %i[ destroy ]

  # POST /writeups
  def create
    game = Game.find(params.expect(:game_id))
    team = Team.find(params.expect(:team_id))

    unless current_user.role == 'admin' || can_manage_team?(team)
      return redirect_to game, alert: 'Недостаточно прав'
    end

    writeup = Writeup.new(
      game: game,
      team: team,
      title: params.expect(:title),
      url: params.expect(:url)
    )

    if writeup.save
      redirect_to game, notice: 'Writeup добавлен.'
    else
      redirect_to game, alert: writeup.errors.full_messages.join(', ')
    end
  end

  # DELETE /writeups/:id
  def destroy
    unless current_user.role == 'admin' || can_manage_team?(@writeup.team)
      return redirect_to @writeup.game, alert: 'Недостаточно прав'
    end
    @writeup.destroy
    redirect_to @writeup.game, notice: 'Writeup удалён.'
  end

  private
    def set_writeup
      @writeup = Writeup.find(params.expect(:id))
    end
end

