# frozen_string_literal: true

require "fileutils"
require "securerandom"
require "zip"
require "digest"

# Управление локальными архивами сервисов (скачивание по URL, сохранение загруженных файлов)
class ServiceArchives
  class Error < StandardError; end
  MAX_ENTRY_BYTES = 200 * 1024 * 1024
  MAX_TOTAL_BYTES = 500 * 1024 * 1024

  # Базовая директория хранения архивов сервисов.
  # Можно переопределить через ENV["SERVICES_STORAGE_DIR"] (например, на проде смонтировать volume).
  ROOT_DIR = begin
    env_dir = ENV["SERVICES_STORAGE_DIR"].to_s.strip
    base = env_dir.present? ? env_dir : Rails.root.join("storage", "services").to_s
    File.expand_path(base)
  end

  # Сохранить загруженный файл (один архив: внутри service/ и опционально checker/)
  def self.save_uploaded(service:, kind:, uploaded_file:)
    new(service).save_uploaded(kind: kind, uploaded_file: uploaded_file)
  end

  # Перескачать архив по URL из поля service_archive_url
  def self.redownload(service:, kind: :service)
    new(service).redownload(kind: kind)
  end

  # Путь до сохранённого локального архива (или nil)
  def self.local_path_for(service, kind)
    case kind.to_sym
    when :service, :archive, :bundle then service.service_local_path
    when :checker then service.checker_local_path
    else nil
    end
  end

  def initialize(service)
    @service = service
    @root_checked = false
  end

  def redownload(kind: :service)
    # Сохраняем совместимость со старым параметром, но реально поддерживаем только один архив.
    kind_sym = kind.to_sym
    kind_sym = :service if kind_sym == :both || kind_sym == :archive || kind_sym == :bundle
    kinds = [ kind_sym ]
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
    kind_sym = kind.to_sym
    kind_sym = :service if kind_sym == :archive || kind_sym == :bundle
    fname = "archive_#{ts}.zip"
    res = ArchiveDownloader.save_uploaded(uploaded_file: uploaded_file, dest_dir: store_dir, filename: fname)
    verify_bundle_file!(res[:path])
    res[:size] = File.size(res[:path])
    res[:sha256] = Digest::SHA256.file(res[:path]).hexdigest
    apply_meta(kind_sym, res)
    @service.save!
    res
  end

  private
  def download_to_local(kind:, url:)
    store_dir = ensure_dir(@service.id)
    ts = Time.current.utc.strftime("%Y%m%d%H%M%S")
    fname = "archive_#{ts}.zip"
    res = ArchiveDownloader.download_url(url: url, dest_dir: store_dir, filename: fname)
    verify_bundle_file!(res[:path])
    res[:size] = File.size(res[:path])
    res[:sha256] = Digest::SHA256.file(res[:path]).hexdigest
    apply_meta(kind, res)
    @service.save!
    res
  end

  def verify_bundle_file!(path)
    has_service = false
    Zip::File.open(path) do |zip|
      zip.each do |e|
        n = e.name.to_s
        next if n.blank?
        has_service ||= n =~ %r{(^|/)service/}
        break if has_service
      end
    end
    return if has_service

    normalize_bundle_zip!(path)

    has_service2 = false
    Zip::File.open(path) do |zip|
      zip.each do |e|
        n = e.name.to_s
        next if n.blank?
        has_service2 ||= n =~ %r{(^|/)service/}
        break if has_service2
      end
    end
    return if has_service2
    FileUtils.rm_f(path)
    raise Error, "в архиве не найден каталог service/"
  rescue Zip::Error => e
    FileUtils.rm_f(path)
    raise Error, "не удалось прочитать zip: #{e.message}"
  end

  def normalize_bundle_zip!(path)
    total = 0
    entries = []
    Zip::File.open(path) do |zip|
      zip.each do |e|
        n = e.name.to_s
        next if n.blank?
        entries << n
      end
    end
    return if entries.empty?

    first_segments = entries.filter_map do |n|
      seg = n.split("/", 2).first
      seg.presence
    end
    common_seg = first_segments.uniq.size == 1 ? first_segments.first : nil
    prefix = common_seg ? "#{common_seg}/" : nil

    tmp = "#{path}.normalized"
    FileUtils.rm_f(tmp)
    Zip::File.open(path) do |src|
      Zip::File.open(tmp, create: true) do |dst|
        src.each do |e|
          name = e.name.to_s
          next if name.blank?
          rel = prefix.present? ? name.sub(/\A#{Regexp.escape(prefix)}/, "") : name.dup
          rel = safe_rel_path(rel)
          next if rel.blank?

          dest_name = if rel == "checker" || rel.start_with?("checker/")
            rel
          else
            "service/#{rel}"
          end

          if name.end_with?("/")
            dst.mkdir(dest_name) unless dst.find_entry(dest_name)
            next
          end

          est = e.size.to_i
          raise Error, "файл в архиве слишком большой (#{est} байт)" if est > MAX_ENTRY_BYTES
          total += est
          raise Error, "архив слишком большой (суммарно #{total} байт)" if total > MAX_TOTAL_BYTES

          dst.get_output_stream(dest_name) do |f|
            written = 0
            e.get_input_stream do |io|
              while (chunk = io.read(16 * 1024))
                break if chunk.empty?
                written += chunk.bytesize
                raise Error, "файл в архиве слишком большой" if written > MAX_ENTRY_BYTES
                f.write(chunk)
              end
            end
          end
        end
      end
    end
    FileUtils.mv(tmp, path)
  rescue Zip::Error, SystemCallError => e
    FileUtils.rm_f(tmp) if tmp
    raise Error, "не удалось нормализовать zip: #{e.message}"
  end

  def safe_rel_path(rel)
    s = rel.to_s.tr("\\", "/")
    return nil if s.empty?
    return nil if s.include?("\0")
    s = s.sub(%r{\A/+}, "")
    s = s.sub(%r{/+\z}, "")
    return nil if s.empty?
    segments = s.split("/")
    return nil if segments.any?(&:blank?)
    return nil if segments.any? { |seg| seg == "." || seg == ".." }
    segments.join("/")
  end

  def apply_meta(kind, res)
    case kind.to_sym
    when :service
      @service.service_local_path = relative_path(res[:path])
      @service.service_local_size = res[:size]
      @service.service_local_sha256 = res[:sha256]
      @service.service_downloaded_at = Time.current
      # Старая схема: сбрасываем checker_* чтобы не путать UI.
      @service.checker_local_path = nil
      @service.checker_local_size = nil
      @service.checker_local_sha256 = nil
      @service.checker_downloaded_at = nil
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
    when :service, :archive, :bundle then @service.service_archive_url.to_s
    when :checker then @service.checker_archive_url.to_s
    else nil
    end
  end

  def ensure_dir(service_id)
    ensure_root_dir!
    dir = File.join(ROOT_DIR, service_id.to_s)
    begin
      FileUtils.mkdir_p(dir)
    rescue SystemCallError => e
      raise Error, "не удалось создать каталог #{dir}: #{e.message}"
    end
    unless File.writable?(dir)
      raise Error, "каталог недоступен для записи: #{dir}. Проверьте права/владельца или задайте SERVICES_STORAGE_DIR"
    end
    dir
  end

  def relative_path(abs)
    abs.to_s.sub(%r{\A#{Regexp.escape(Rails.root.to_s)}/?}, "")
  end

  def ensure_root_dir!
    return if @root_checked
    begin
      FileUtils.mkdir_p(ROOT_DIR)
    rescue SystemCallError => e
      raise Error, "не удалось создать базовый каталог #{ROOT_DIR}: #{e.message}"
    end
    unless File.directory?(ROOT_DIR) && File.writable?(ROOT_DIR)
      raise Error, "базовый каталог недоступен для записи: #{ROOT_DIR}. Проверьте права или задайте SERVICES_STORAGE_DIR"
    end
    @root_checked = true
  end
end
