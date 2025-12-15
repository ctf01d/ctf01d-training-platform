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
    unless @service.checker_archive_url.present?
      return redirect_to @service, alert: "Не указан URL архива чекера."
    end
    # Заглушка: имитируем отправку задачи на проверку чекера
    @service.update(check_status: "queued", checked_at: Time.current)
    redirect_to @service, notice: "Проверка чекера запущена (заглушка): статус queued."
  end

  # POST /services/:id/redownload
  # Переcкачать архивы по указанным URL с валидацией zip
  def redownload
    what = params[:what].to_s.presence || "both"
    kind = case what
    when "service" then :service
    when "checker" then :checker
    else :both
    end
    results = ServiceArchives.redownload(service: @service, kind: kind)
    if results.empty?
      redirect_to @service, alert: "Нет URL для скачивания (service/checker)."
    else
      redirect_to @service, notice: "Архивы перескачаны: #{results.keys.join(', ')}."
    end
  rescue ArchiveDownloader::Error, ServiceArchives::Error => e
    redirect_to @service, alert: "Ошибка скачивания: #{e.message}"
  end

  # POST /services/:id/upload_archives
  # Загрузка локальных zip файлов (service/checker)
  def upload_archives
    uploaded = []
    svc_file = params.dig(:service, :service_archive_file)
    chk_file = params.dig(:service, :checker_archive_file)
    if svc_file
      ServiceArchives.save_uploaded(service: @service, kind: :service, uploaded_file: svc_file)
      uploaded << "service"
    end
    if chk_file
      ServiceArchives.save_uploaded(service: @service, kind: :checker, uploaded_file: chk_file)
      uploaded << "checker"
    end
    if uploaded.empty?
      redirect_to @service, alert: "Не выбраны файлы для загрузки."
    else
      redirect_to @service, notice: "Загружено: #{uploaded.join(', ')}."
    end
  rescue ArchiveDownloader::Error, ServiceArchives::Error => e
    redirect_to @service, alert: "Ошибка загрузки: #{e.message}"
  end

  # GET /services/:id/download_local?what=service|checker
  def download_local
    what = params[:what].to_s
    path_rel = case what
    when "service" then @service.service_local_path
    when "checker" then @service.checker_local_path
    else nil
    end
    return redirect_to @service, alert: "Не найден локальный архив." if path_rel.blank?
    abs = path_rel.to_s.start_with?("/") ? path_rel.to_s : Rails.root.join(path_rel)
    unless File.file?(abs)
      return redirect_to @service, alert: "Файл отсутствует на диске."
    end
    send_file abs, filename: File.basename(abs), type: "application/zip"
  end

  # GET /services/import_github
  def import_github
    require_admin
    if request.get?
      # форма импорта
    else
      repo_url = params.expect(:repo_url)
      ref = params[:ref]
      data = GithubImporter.import(repo_url: repo_url, ref: ref)
      svc = Service.new(
        name: data[:name],
        author: data[:author],
        public_description: data[:public_description],
        copyright: begin
          cr = data[:copyright]
          if data[:license].present?
            suffix = "License: #{data[:license]}"
            cr.present? ? "#{cr} - #{suffix}" : suffix
          else
            cr
          end
        end,
        public: false
      )
      if svc.save
        # Сохраним сформированные архивы
        if data[:archives][:service].present?
          uploaded_like = uploaded_from_bytes(data[:archives][:service], "service.zip")
          ServiceArchives.save_uploaded(service: svc, kind: :service, uploaded_file: uploaded_like)
        end
        if data[:archives][:checker].present?
          uploaded_like = uploaded_from_bytes(data[:archives][:checker], "checker.zip")
          ServiceArchives.save_uploaded(service: svc, kind: :checker, uploaded_file: uploaded_like)
        end
        redirect_to svc, notice: "Сервис импортирован из GitHub."
      else
        @errors = svc.errors.full_messages
        render :import_github, status: :unprocessable_content
      end
    end
  rescue GithubImporter::Error, ArchiveDownloader::Error, ServiceArchives::Error => e
    @errors = [ e.message ]
    render :import_github, status: :unprocessable_content
  end

  # GET/POST /services/import_zip
  def import_zip
    require_admin
    if request.get?
      return
    end

    upload = params[:archive]
    raise ArgumentError, "файл не выбран" unless upload.respond_to?(:read)
    data = upload.read
    raise ArgumentError, "пустой файл" if data.blank?

    parts = split_zip(data)
    service_zip = parts[:service]
    checker_zip = parts[:checker]
    raise ArgumentError, "не найден каталог service/ или содержимое архива" if service_zip.nil?

    name = params[:name].presence || File.basename(upload.original_filename.to_s, File.extname(upload.original_filename.to_s)).tr("_", " ").strip
    author = current_user&.display_name || "upload"

    svc = Service.new(
      name: name.presence || "Uploaded service",
      author: author,
      public_description: params[:description],
      public: false
    )

    if svc.save
      ServiceArchives.save_uploaded(service: svc, kind: :service, uploaded_file: uploaded_from_bytes(service_zip, "service.zip"))
      if checker_zip
        ServiceArchives.save_uploaded(service: svc, kind: :checker, uploaded_file: uploaded_from_bytes(checker_zip, "checker.zip"))
      end
      redirect_to svc, notice: "Сервис импортирован из ZIP."
    else
      @errors = svc.errors.full_messages
      render :import_zip, status: :unprocessable_content
    end
  rescue Zip::Error, ArgumentError, ArchiveDownloader::Error, ServiceArchives::Error => e
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
                               :service_archive_url, :checker_archive_url, :writeup_url, :exploits_url ])
    end

    def uploaded_from_bytes(bytes, filename)
      # Мини-объект, похожий на UploadedFile
      StringIO.new(bytes).tap do |io|
        io.define_singleton_method(:original_filename) { filename }
        io.define_singleton_method(:content_type) { "application/zip" }
        io.define_singleton_method(:size) { bytes.bytesize }
      end
    end

    def split_zip(zip_bytes)
      buffer = StringIO.new(zip_bytes)
      service_zip = nil
      checker_zip = nil
      root_prefix = nil

      Zip::File.open_buffer(buffer) do |zip|
        first = zip.first
        root_prefix = if first&.name&.include?("/")
          first.name.split("/").first + "/"
        else
          ""
        end

        # если есть каталоги service/ и checker/
        has_service = zip.any? { |e| e.name.start_with?(File.join(root_prefix, "service/")) }
        has_checker = zip.any? { |e| e.name.start_with?(File.join(root_prefix, "checker/")) }

        service_zip = build_subzip(zip_bytes, File.join(root_prefix, has_service ? "service/" : ""))
        checker_zip = build_subzip(zip_bytes, File.join(root_prefix, "checker/")) if has_checker
      end

      { service: service_zip, checker: checker_zip }
    end

    def build_subzip(zip_bytes, subdir_prefix)
      buffer = StringIO.new
      Zip::OutputStream.write_buffer(buffer) do |zos|
        Zip::File.open_buffer(StringIO.new(zip_bytes)) do |zip|
          zip.each do |entry|
            next unless entry.name.start_with?(subdir_prefix)
            rel = entry.name.sub(subdir_prefix, "")
            next if rel.empty?
            if entry.name.end_with?("/")
              zos.put_next_entry(rel + "/")
            else
              zos.put_next_entry(rel)
              zos.write(entry.get_input_stream.read)
            end
          end
        end
      end
      buffer.rewind
      buffer.read
    end
end
