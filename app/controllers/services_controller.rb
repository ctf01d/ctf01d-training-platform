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
    # Заглушка: имитируем отправку задачи на проверку чекера
    @service.update(check_status: "queued", checked_at: Time.current)
    redirect_to @service, notice: "Проверка запущена (заглушка): статус queued."
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
    path_rel = @service.service_local_path
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
        if data.dig(:archives, :bundle).present?
          uploaded_like = uploaded_from_bytes(data[:archives][:bundle], "archive.zip")
          ServiceArchives.save_uploaded(service: svc, kind: :service, uploaded_file: uploaded_like)
        end
        if data[:archive_url].present?
          svc.update(service_archive_url: data[:archive_url])
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
    service_part = parts[:service]
    checker_part = parts[:checker]
    raise ArgumentError, "не найден каталог service/ или содержимое архива" if service_part.nil?

    name = params[:name].presence || File.basename(upload.original_filename.to_s, File.extname(upload.original_filename.to_s)).tr("_", " ").strip
    author = current_user&.display_name || "upload"

    svc = Service.new(
      name: name.presence || "Uploaded service",
      author: author,
      public_description: params[:description],
      public: false
    )

    if svc.save
      bundle = build_bundle_zip(service_part: service_part, checker_part: checker_part)
      ServiceArchives.save_uploaded(service: svc, kind: :service, uploaded_file: uploaded_from_bytes(bundle, "archive.zip"))
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

    def build_bundle_zip(service_part:, checker_part:)
      buffer = StringIO.new
      Zip::OutputStream.write_buffer(buffer) do |zos|
        copy_zip_entries(zos, service_part, "service/")
        copy_zip_entries(zos, checker_part, "checker/") if checker_part.present?
      end
      buffer.rewind
      buffer.read
    end

    def copy_zip_entries(zos, zip_bytes, prefix)
      Zip::File.open_buffer(StringIO.new(zip_bytes)) do |zip|
        zip.each do |entry|
          next if entry.name.to_s.strip.empty?
          name = entry.name.to_s.sub(%r{\A/+}, "")
          if entry.directory?
            dir_name = name.end_with?("/") ? name : "#{name}/"
            zos.put_next_entry(prefix + dir_name)
          else
            zos.put_next_entry(prefix + name)
            zos.write(entry.get_input_stream.read)
          end
        end
      end
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
