class ScoreboardsController < ApplicationController
  # Public scoreboard
  def index
    @games = Game.includes(:final_results, results: :team).order(starts_at: :desc, created_at: :desc)
  end
end
