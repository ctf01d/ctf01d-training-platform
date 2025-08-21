class GamesController < ApplicationController
  before_action :require_admin, except: %i[index show]
  before_action :set_game, only: %i[ show edit update destroy ]
  before_action :set_game_for_manage, only: %i[ manage_services add_service remove_service ]

  # GET /games
  def index
    @ongoing_games  = Game.ongoing.order(ends_at: :asc)
    @upcoming_games = Game.upcoming.order(starts_at: :asc)
    @past_games     = Game.past.order(ends_at: :desc)
  end

  # GET /games/1
  def show
    if @game.final_results.exists?
      @final = true
      @results = @game.final_results.includes(:team).order(position: :asc, score: :desc)
    else
      @final = false
      @results = @game.results.includes(:team).order(score: :desc)
    end
    @writeups = @game.writeups.includes(:team).order(created_at: :desc)
    if user_signed_in?
      # команды пользователя, которыми он может управлять
      my_team_ids = TeamMembership.where(user_id: current_user.id, status: TeamMembership::STATUS_APPROVED).pluck(:team_id)
      @my_manageable_teams = Team.where(id: my_team_ids).select { |t| can_manage_team?(t) }
    else
      @my_manageable_teams = []
    end
    @can_access = can_access_game?(@game)
  end

  # GET /games/:id/manage_services
  def manage_services
    @services = Service.order(:name)
    @attached_ids = @game.services.pluck(:id).to_set
  end

  # POST /games/:id/add_service
  def add_service
    service = Service.find(params.expect(:service_id))
    @game.services << service unless @game.services.exists?(service.id)
    redirect_to manage_services_game_path(@game), notice: 'Сервис добавлен.'
  end

  # DELETE /games/:id/remove_service
  def remove_service
    service = Service.find(params.expect(:service_id))
    @game.services.destroy(service)
    redirect_to manage_services_game_path(@game), notice: 'Сервис удалён.'
  end

  # GET /games/new
  def new
    @game = Game.new
  end

  # GET /games/1/edit
  def edit
  end

  # POST /games
  def create
    @game = Game.new(game_params)

    if @game.save
      redirect_to @game, notice: "Игра создана."
    else
      render :new, status: :unprocessable_entity
    end
  end

  # PATCH/PUT /games/1
  def update
    if @game.update(game_params)
      redirect_to @game, notice: "Игра обновлена.", status: :see_other
    else
      render :edit, status: :unprocessable_entity
    end
  end

  # DELETE /games/1
  def destroy
    @game.destroy!
    redirect_to games_path, notice: "Игра удалена.", status: :see_other
  end

  # POST /games/:id/finalize
  def finalize
    game = Game.find(params.expect(:id))
    ActiveRecord::Base.transaction do
      # Prevent double-finalization
      if game.final_results.exists?
        game.update!(finalized: true, finalized_at: Time.current) unless game.finalized
      else
        rows = game.results.includes(:team).order(score: :desc).to_a
        rows.each_with_index do |r, idx|
          FinalResult.create!(game_id: game.id, team_id: r.team_id, score: r.score.to_i, position: idx + 1)
        end
        game.update!(finalized: true, finalized_at: Time.current)
      end
    end
    redirect_to game, notice: 'Итоги зафиксированы.'
  rescue => e
    redirect_to game, alert: 'Не удалось зафиксировать итоги.'
  end

  # DELETE /games/:id/unfinalize
  def unfinalize
    game = Game.find(params.expect(:id))
    ActiveRecord::Base.transaction do
      game.final_results.delete_all
      game.update!(finalized: false, finalized_at: nil)
    end
    redirect_to game, notice: 'Финализация снята.'
  rescue => e
    redirect_to game, alert: 'Не удалось снять финализацию.'
  end

  private
    # Use callbacks to share common setup or constraints between actions.
    def set_game
      @game = Game.find(params.expect(:id))
    end

    def set_game_for_manage
      require_admin
      @game = Game.find(params.expect(:id))
    end

    # Only allow a list of trusted parameters through.
  def game_params
      params.expect(game: [ :name, :organizer, :starts_at, :ends_at, :avatar_url, :site_url, :ctftime_url,
                            :registration_opens_at, :registration_closes_at,
                            :scoreboard_opens_at, :scoreboard_closes_at,
                            :vpn_url, :vpn_config_url, :access_secret, :access_instructions,
                            { service_ids: [] } ])
    end
end
