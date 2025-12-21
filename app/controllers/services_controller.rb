class ServicesController < ApplicationController
  require "zip"
  before_action :require_admin, except: %i[index show]
  before_action :set_service, only: %i[ show edit update destroy toggle_public check_checker redownload upload_archives download_local ]

  # GET /services
  def index
    @services = if current_user&.role == "admin"
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
      redirect_to @service, notice: "Сервис создан."
    else
      render :new, status: :unprocessable_content
    end
  end

  # PATCH/PUT /services/1
  def update
    if @service.update(service_params)
      redirect_to @service, notice: "Сервис обновлён.", status: :see_other
    else
      render :edit, status: :unprocessable_content
    end
  end

  # DELETE /services/1
  def destroy
    @service.destroy!
    redirect_to services_path, notice: "Сервис удалён.", status: :see_other
  end

  # POST /services/:id/toggle_public
  def toggle_public
    @service.update(public: !@service.public)
    status = @service.public ? "публичным" : "приватным"
    redirect_to services_path, notice: "Сервис стал #{status}."
  end

  # POST /services/:id/check_checker
  def check_checker
    unless @service.service_archive_url.present? || @service.service_local_path.present?
      return redirect_to @service, alert: "Не указан URL архива."
    end

    zip_path = ensure_service_bundle_path!(@service)
    res = CheckerInspector.call(zip_path: zip_path)
    @service.update(check_status: res[:status], checked_at: Time.current)

    msg = case res[:status]
    when "missing" then "Чекер: отсутствует."
    when "present" then "Чекер: есть."
    when "codes" then "Чекер: есть, (присутствуют коды ответа 101-104)."
    else "Проверка выполнена: #{res[:status]}."
    end
    redirect_to @service, notice: msg
  rescue CheckerInspector::Error, ServiceArchives::Error => e
    @service.update(check_status: "error", checked_at: Time.current)
    redirect_to @service, alert: "Ошибка проверки: #{e.message}"
  end

  # POST /services/:id/redownload
  # Переcкачать архивы по указанным URL с валидацией zip
  def redownload
    results = ServiceArchives.redownload(service: @service, kind: :service)
    if results.empty?
      redirect_to @service, alert: "Нет URL для скачивания архива."
    else
      redirect_to @service, notice: "Архив перескачан."
    end
  rescue ArchiveDownloader::Error, ServiceArchives::Error => e
    redirect_to @service, alert: "Ошибка скачивания: #{e.message}"
  end

  # POST /services/:id/upload_archives
  # Загрузка локального zip архива (внутри service/ и опционально checker/)
  def upload_archives
    archive_file = params.dig(:service, :archive_file)
    return redirect_to @service, alert: "Не выбран файл для загрузки." unless archive_file
    ServiceArchives.save_uploaded(service: @service, kind: :service, uploaded_file: archive_file)
    redirect_to @service, notice: "Архив загружен."
  rescue ArchiveDownloader::Error, ServiceArchives::Error => e
    redirect_to @service, alert: "Ошибка загрузки: #{e.message}"
  end

  # GET /services/:id/download_local?what=service|checker
  def download_local
    store_dir = File.join(ServiceArchives::ROOT_DIR, @service.id.to_s)
    candidates = Dir.glob(File.join(store_dir, "*.zip"))
    return redirect_to @service, alert: "Не найден локальный архив." if candidates.empty?

    abs = candidates.max_by { |p| File.mtime(p) }
    return redirect_to @service, alert: "Файл отсутствует на диске." unless abs && File.file?(abs)

    send_file abs, filename: File.basename(abs), type: "application/zip"
  end

  # GET /services/import_github
  def import_github
    require_admin
    if request.get? || request.head?
      # форма импорта
    else
      repo_url = params.expect(:repo_url)
      ref = params[:ref]
      fetch = GithubImporter.fetch(repo_url: repo_url, ref: ref)
      if fetch[:zip_bytes].bytesize > 120 * 1024 * 1024
        raise GithubImporter::Error, "архив репозитория слишком большой"
      end
      bundle = ServiceImport::BundleBuilder.call(zip_bytes: fetch[:zip_bytes])
      meta = ServiceImport::MetadataExtractor.call(bundle_zip_bytes: bundle)
      svc = Service.new(
        name: meta[:name].presence || fetch[:repo].to_s.tr("-", " ").strip,
        author: fetch[:owner],
        public_description: meta[:public_description],
        copyright: begin
          cr = meta[:copyright]
          if meta[:license].present?
            suffix = "License: #{meta[:license]}"
            cr.present? ? "#{cr} - #{suffix}" : suffix
          else
            cr
          end
        end,
        public: false
      )
      if svc.save
        # Сохраним сформированные архивы
        uploaded_like = uploaded_from_bytes(bundle, "archive.zip")
        ServiceArchives.save_uploaded(service: svc, kind: :service, uploaded_file: uploaded_like)
        svc.update(service_archive_url: fetch[:archive_url]) if fetch[:archive_url].present?
        redirect_to svc, notice: "Сервис импортирован из GitHub."
      else
        @errors = svc.errors.full_messages
        render :import_github, status: :unprocessable_content
      end
    end
  rescue GithubImporter::Error, ServiceImport::BundleBuilder::Error, ArchiveDownloader::Error, ServiceArchives::Error => e
    @errors = [ e.message ]
    render :import_github, status: :unprocessable_content
  end

  # GET/POST /services/import_zip
  def import_zip
    require_admin
    if request.get? || request.head?
      return
    end

    upload = params[:archive]
    raise ArgumentError, "файл не выбран" unless upload.respond_to?(:read)
    max_bytes = 120 * 1024 * 1024
    if upload.respond_to?(:size) && upload.size.to_i > max_bytes
      raise ArgumentError, "архив слишком большой"
    end
    data = upload.read(max_bytes + 1)
    raise ArgumentError, "архив слишком большой" if data.bytesize > max_bytes
    raise ArgumentError, "пустой файл" if data.blank?

    bundle = ServiceImport::BundleBuilder.call(zip_bytes: data)
    meta = ServiceImport::MetadataExtractor.call(bundle_zip_bytes: bundle)

    name = params[:name].presence ||
      meta[:name].presence ||
      File.basename(upload.original_filename.to_s, File.extname(upload.original_filename.to_s)).tr("_", " ").strip
    author = current_user&.display_name || "upload"
    svc = Service.new(
      name: name.presence || "Uploaded service",
      author: author,
      public_description: params[:description].presence || meta[:public_description],
      copyright: begin
        cr = meta[:copyright]
        if meta[:license].present?
          suffix = "License: #{meta[:license]}"
          cr.present? ? "#{cr} - #{suffix}" : suffix
        else
          cr
        end
      end,
      public: false
    )

    if svc.save
      ServiceArchives.save_uploaded(service: svc, kind: :service, uploaded_file: uploaded_from_bytes(bundle, "archive.zip"))
      redirect_to svc, notice: "Сервис импортирован из ZIP."
    else
      @errors = svc.errors.full_messages
      render :import_zip, status: :unprocessable_content
    end
  rescue ServiceImport::BundleBuilder::Error, Zip::Error, ArgumentError, ArchiveDownloader::Error, ServiceArchives::Error => e
    @errors = [ e.message ]
    render :import_zip, status: :unprocessable_content
  end

  private
    # Use callbacks to share common setup or constraints between actions.
    def set_service
      @service = Service.find(params.expect(:id))
    end

    # Only allow a list of trusted parameters through.
    def service_params
      params.expect(service: [ :name, :public_description, :private_description, :author, :copyright, :avatar_url, :public,
                               :service_archive_url, :writeup_url, :exploits_url ])
    end

    def uploaded_from_bytes(bytes, filename)
      # Мини-объект, похожий на UploadedFile
      StringIO.new(bytes).tap do |io|
        io.define_singleton_method(:original_filename) { filename }
        io.define_singleton_method(:content_type) { "application/zip" }
        io.define_singleton_method(:size) { bytes.bytesize }
      end
    end

    def ensure_service_bundle_path!(service)
      rel = service.service_local_path.to_s
      abs = if rel.present?
        rel.start_with?("/") ? rel : Rails.root.join(rel).to_s
      else
        nil
      end
      return abs if abs && File.file?(abs)

      if service.service_archive_url.present?
        ServiceArchives.redownload(service: service, kind: :service)
        rel = service.service_local_path.to_s
        abs = rel.present? ? Rails.root.join(rel).to_s : nil
        return abs if abs && File.file?(abs)
      end

      raise ServiceArchives::Error, "не найден локальный архив и отсутствует Archive URL"
    end
end
