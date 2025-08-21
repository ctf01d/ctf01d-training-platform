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
    @results = @game.results.includes(:team).order(score: :desc)
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
      params.expect(game: [ :name, :organizer, :starts_at, :ends_at, :avatar_url, :site_url, :ctftime_url, { service_ids: [] } ])
    end
end
