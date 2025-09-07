class UniversitiesController < ApplicationController
  before_action :require_admin, except: %i[index show]
  before_action :set_university, only: %i[ show edit update destroy ]

  # GET /universities
  def index
    @universities = University.all
  end

  # GET /universities/1
  def show
  end

  # GET /universities/new
  def new
    @university = University.new
  end

  # GET /universities/1/edit
  def edit
  end

  # POST /universities
  def create
    @university = University.new(university_params)

    if @university.save
      redirect_to @university, notice: "Университет создан."
    else
      render :new, status: :unprocessable_content
    end
  end

  # PATCH/PUT /universities/1
  def update
    if @university.update(university_params)
      redirect_to @university, notice: "Университет обновлён.", status: :see_other
    else
      render :edit, status: :unprocessable_content
    end
  end

  # DELETE /universities/1
  def destroy
    @university.destroy!
    redirect_to universities_path, notice: "Университет удалён.", status: :see_other
  end

  private
    # Use callbacks to share common setup or constraints between actions.
    def set_university
      @university = University.find(params.expect(:id))
    end

    # Only allow a list of trusted parameters through.
    def university_params
      params.expect(university: [ :name, :site_url, :avatar_url ])
    end
end
