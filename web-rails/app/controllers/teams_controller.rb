class TeamsController < ApplicationController
  before_action :require_admin, except: %i[index show]
  before_action :set_team, only: %i[ show edit update destroy ]

  # GET /teams
  def index
    @teams = Team.all
  end

  # GET /teams/1
  def show
  end

  # GET /teams/new
  def new
    @team = Team.new
  end

  # GET /teams/1/edit
  def edit
  end

  # POST /teams
  def create
    @team = Team.new(team_params)

    if @team.save
      redirect_to @team, notice: "Team was successfully created."
    else
      render :new, status: :unprocessable_entity
    end
  end

  # PATCH/PUT /teams/1
  def update
    if @team.update(team_params)
      redirect_to @team, notice: "Team was successfully updated.", status: :see_other
    else
      render :edit, status: :unprocessable_entity
    end
  end

  # DELETE /teams/1
  def destroy
    @team.destroy!
    redirect_to teams_path, notice: "Team was successfully destroyed.", status: :see_other
  end

  private
    # Use callbacks to share common setup or constraints between actions.
    def set_team
      @team = Team.find(params.expect(:id))
    end

    # Only allow a list of trusted parameters through.
    def team_params
      params.expect(team: [ :name, :university, :description, :website, :avatar_url, :captain_id ])
    end
end
