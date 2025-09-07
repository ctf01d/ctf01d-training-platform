# frozen_string_literal: true

require "fileutils"
require "securerandom"

# Управление локальными архивами сервисов (скачивание по URL, сохранение загруженных файлов)
class ServiceArchives
  class Error < StandardError; end

  ROOT_DIR = Rails.root.join("storage", "services").to_s

  # Сохранить загруженный файл (service|checker)
  def self.save_uploaded(service:, kind:, uploaded_file:)
    new(service).save_uploaded(kind: kind, uploaded_file: uploaded_file)
  end

  # Перескачать по URL из полей модели (service_archive_url|checker_archive_url)
  def self.redownload(service:, kind: :both)
    new(service).redownload(kind: kind)
  end

  # Путь до сохранённого локального архива (или nil)
  def self.local_path_for(service, kind)
    case kind.to_sym
    when :service then service.service_local_path
    when :checker then service.checker_local_path
    else nil
    end
  end

  def initialize(service)
    @service = service
  end

  def redownload(kind: :both)
    kinds = (kind == :both ? [ :service, :checker ] : [ kind.to_sym ])
    results = {}
    kinds.each do |k|
      url = url_for(k)
      next if url.blank?
      results[k] = download_to_local(kind: k, url: url)
    end
    results
  end

  def save_uploaded(kind:, uploaded_file:)
    store_dir = ensure_dir(@service.id)
    ts = Time.current.utc.strftime("%Y%m%d%H%M%S")
    fname = "#{kind}_#{ts}.zip"
    res = ArchiveDownloader.save_uploaded(uploaded_file: uploaded_file, dest_dir: store_dir, filename: fname)
    apply_meta(kind, res)
    @service.save!
    res
  end

  private
  def download_to_local(kind:, url:)
    store_dir = ensure_dir(@service.id)
    ts = Time.current.utc.strftime("%Y%m%d%H%M%S")
    fname = "#{kind}_#{ts}.zip"
    res = ArchiveDownloader.download_url(url: url, dest_dir: store_dir, filename: fname)
    apply_meta(kind, res)
    @service.save!
    res
  end

  def apply_meta(kind, res)
    case kind.to_sym
    when :service
      @service.service_local_path = relative_path(res[:path])
      @service.service_local_size = res[:size]
      @service.service_local_sha256 = res[:sha256]
      @service.service_downloaded_at = Time.current
    when :checker
      @service.checker_local_path = relative_path(res[:path])
      @service.checker_local_size = res[:size]
      @service.checker_local_sha256 = res[:sha256]
      @service.checker_downloaded_at = Time.current
    else
      raise Error, "неизвестный тип архива: #{kind}"
    end
  end

  def url_for(kind)
    case kind.to_sym
    when :service then @service.service_archive_url.to_s
    when :checker then @service.checker_archive_url.to_s
    else nil
    end
  end

  def ensure_dir(service_id)
    dir = File.join(ROOT_DIR, service_id.to_s)
    FileUtils.mkdir_p(dir)
    dir
  end

  def relative_path(abs)
    abs.to_s.sub(%r{\A#{Regexp.escape(Rails.root.to_s)}/?}, "")
  end
end
