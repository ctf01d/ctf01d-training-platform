class TeamMembershipsController < ApplicationController
  before_action :require_admin, except: %i[index show]
  before_action :set_team_membership, only: %i[ show edit update destroy ]

  # GET /team_memberships
  def index
    @team_memberships = TeamMembership.all
  end

  # GET /team_memberships/1
  def show
  end

  # GET /team_memberships/new
  def new
    @team_membership = TeamMembership.new
  end

  # GET /team_memberships/1/edit
  def edit
  end

  # POST /team_memberships
  def create
    @team_membership = TeamMembership.new(team_membership_params)

    if @team_membership.save
      redirect_to @team_membership, notice: "Team membership was successfully created."
    else
      render :new, status: :unprocessable_entity
    end
  end

  # PATCH/PUT /team_memberships/1
  def update
    if @team_membership.update(team_membership_params)
      redirect_to @team_membership, notice: "Team membership was successfully updated.", status: :see_other
    else
      render :edit, status: :unprocessable_entity
    end
  end

  # DELETE /team_memberships/1
  def destroy
    @team_membership.destroy!
    redirect_to team_memberships_path, notice: "Team membership was successfully destroyed.", status: :see_other
  end

  private
    # Use callbacks to share common setup or constraints between actions.
    def set_team_membership
      @team_membership = TeamMembership.find(params.expect(:id))
    end

    # Only allow a list of trusted parameters through.
    def team_membership_params
      params.expect(team_membership: [ :team_id, :user_id, :role, :status ])
    end
end
