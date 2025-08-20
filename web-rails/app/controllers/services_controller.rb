class ServicesController < ApplicationController
  before_action :require_admin, except: %i[index show]
  before_action :set_service, only: %i[ show edit update destroy toggle_public ]

  # GET /services
  def index
    @services = if current_user&.role == 'admin'
      Service.order(public: :desc, name: :asc)
    else
      Service.publicly_visible.order(name: :asc)
    end
  end

  # GET /services/1
  def show
  end

  # GET /services/new
  def new
    @service = Service.new
  end

  # GET /services/1/edit
  def edit
  end

  # POST /services
  def create
    @service = Service.new(service_params)

    if @service.save
      redirect_to @service, notice: "Service was successfully created."
    else
      render :new, status: :unprocessable_entity
    end
  end

  # PATCH/PUT /services/1
  def update
    if @service.update(service_params)
      redirect_to @service, notice: "Service was successfully updated.", status: :see_other
    else
      render :edit, status: :unprocessable_entity
    end
  end

  # DELETE /services/1
  def destroy
    @service.destroy!
    redirect_to services_path, notice: "Service was successfully destroyed.", status: :see_other
  end

  # POST /services/:id/toggle_public
  def toggle_public
    @service.update(public: !@service.public)
    status = @service.public ? 'публичным' : 'приватным'
    redirect_to services_path, notice: "Сервис стал #{status}."
  end

  # POST /services/:id/check_checker
  def check_checker
    unless @service.checker_archive_url.present?
      return redirect_to @service, alert: 'Не указан URL архива чекера.'
    end
    # Заглушка: имитируем отправку задачи на проверку чекера
    redirect_to @service, notice: 'Проверка чекера запущена (заглушка).'
  end

  private
    # Use callbacks to share common setup or constraints between actions.
    def set_service
      @service = Service.find(params.expect(:id))
    end

    # Only allow a list of trusted parameters through.
    def service_params
      params.expect(service: [ :name, :public_description, :private_description, :author, :copyright, :avatar_url, :public,
                               :service_archive_url, :checker_archive_url, :writeup_url ])
    end
end
