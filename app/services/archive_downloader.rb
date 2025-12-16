# frozen_string_literal: true

require "net/http"
require "uri"
require "digest"
require "fileutils"
require "openssl"

# Утилита для скачивания и проверки zip-архивов по URL или из загруженного файла
class ArchiveDownloader
  class Error < StandardError; end

  # Скачать по URL в каталог `dest_dir` с именем `filename` (опционально)
  # Возвращает хеш: { path:, size:, sha256:, content_type: }
  def self.download_url(url:, dest_dir:, filename: nil, open_timeout: 5, read_timeout: 30, max_bytes: 200 * 1024 * 1024)
    new.download_url(url: url, dest_dir: dest_dir, filename: filename, open_timeout: open_timeout, read_timeout: read_timeout, max_bytes: max_bytes)
  end

  # Сохранить загруженный файл (ActionDispatch::Http::UploadedFile) в каталог `dest_dir`
  # Возвращает хеш: { path:, size:, sha256:, content_type: }
  def self.save_uploaded(uploaded_file:, dest_dir:, filename: nil, max_bytes: 200 * 1024 * 1024)
    new.save_uploaded(uploaded_file: uploaded_file, dest_dir: dest_dir, filename: filename, max_bytes: max_bytes)
  end

  def download_url(url:, dest_dir:, filename:, open_timeout:, read_timeout:, max_bytes:)
    raise Error, "пустой URL" if url.to_s.strip.empty?
    uri = URI.parse(url)
    raise Error, "поддерживаются только http(s)://" unless %w[http https].include?(uri.scheme)

    redirects = 0
    content_type = nil

    loop do
      raise Error, "слишком много редиректов" if redirects > 5

      http = Net::HTTP.new(uri.host, uri.port)
      http.use_ssl = (uri.scheme == "https")
      http.open_timeout = open_timeout
      http.read_timeout = read_timeout
      configure_ssl!(http) if http.use_ssl?

      # HEAD для быстрого определения типа/размера, если поддерживается
      begin
        head = Net::HTTP::Head.new(uri.request_uri)
        head_res = http.request(head)
      rescue OpenSSL::SSL::SSLError
        head_res = nil
      rescue StandardError
        head_res = nil
      end

      content_type = head_res&.[]("content-type")&.split(";")&.first
      content_len = head_res&.[]("content-length")&.to_i
      if content_len && content_len > 0 && content_len > max_bytes
        raise Error, "слишком большой архив: #{content_len} байт"
      end

      # GET загрузка
      req = Net::HTTP::Get.new(uri.request_uri)
      res = http.request(req)
      if res.is_a?(Net::HTTPRedirection)
        location = res["location"].to_s
        raise Error, "редирект без Location" if location.blank?
        uri = URI.join(uri.to_s, location)
        redirects += 1
        next
      end
      unless res.is_a?(Net::HTTPSuccess)
        raise Error, "не удалось скачать: HTTP #{res.code}"
      end
      content_type ||= res["content-type"]&.split(";")&.first

      # Проверка что это zip по типу/сигнатуре
      body = res.body
      if body.bytesize > max_bytes
        raise Error, "слишком большой архив: > #{max_bytes} байт"
      end
      unless zip_payload?(body, content_type)
        raise Error, "ожидался zip-архив, получен #{content_type || 'unknown'}"
      end

      FileUtils.mkdir_p(dest_dir)
      fname = filename || safe_filename_from(uri)
      path = File.join(dest_dir, fname)
      File.binwrite(path, body)
      return { path: path, size: File.size(path), sha256: Digest::SHA256.file(path).hexdigest, content_type: content_type }
    end
  rescue OpenSSL::SSL::SSLError => e
    raise Error, "SSL ошибка при скачивании: #{e.message}"
  rescue Net::OpenTimeout, Net::ReadTimeout => e
    raise Error, "таймаут при скачивании: #{e.message}"
  end

  def save_uploaded(uploaded_file:, dest_dir:, filename:, max_bytes:)
    raise Error, "файл не передан" unless uploaded_file
    io = uploaded_file.respond_to?(:read) ? uploaded_file : nil
    raise Error, "неподдерживаемый тип файла" unless io

    # Прочитаем в память до лимита (или используем tempfile.path если есть)
    if uploaded_file.respond_to?(:size) && uploaded_file.size.to_i > max_bytes
      raise Error, "слишком большой архив: > #{max_bytes} байт"
    end

    content_type = (uploaded_file.respond_to?(:content_type) && uploaded_file.content_type).to_s.split(";").first
    raw = io.read
    unless zip_payload?(raw, content_type)
      raise Error, "ожидался zip-архив, получен #{content_type.presence || 'unknown'}"
    end

    FileUtils.mkdir_p(dest_dir)
    fname = filename || sanitize_filename(uploaded_file.respond_to?(:original_filename) ? uploaded_file.original_filename : "archive.zip")
    fname = ensure_zip_ext(fname)
    path = File.join(dest_dir, fname)
    File.binwrite(path, raw)
    { path: path, size: File.size(path), sha256: Digest::SHA256.file(path).hexdigest, content_type: content_type }
  end

  private
  def zip_payload?(bytes, content_type)
    return false unless bytes && bytes.bytesize >= 4
    # ZIP magic: PK\x03\x04, PK\x05\x06 (empty), PK\x07\x08
    sig = bytes.bytes.first(4)
    is_zip_magic = sig[0] == 0x50 && sig[1] == 0x4B
    return true if is_zip_magic
    # иногда сервер ставит generic octet-stream
    %w[application/zip application/x-zip-compressed application/octet-stream].include?(content_type.to_s)
  end

  def sanitize_filename(name)
    base = File.basename(name.to_s)
    base.gsub(/[^a-zA-Z0-9_.-]+/, "_")
  end

  def ensure_zip_ext(name)
    File.extname(name).downcase == ".zip" ? name : "#{name}.zip"
  end

  def safe_filename_from(uri)
    guess = File.basename(uri.path.presence || "archive.zip")
    guess = ensure_zip_ext(guess)
    sanitize_filename(guess)
  end

  def configure_ssl!(http)
    store = OpenSSL::X509::Store.new
    store.set_default_paths
    # В некоторых окружениях включена проверка CRL на уровне openssl.cnf,
    # что ломает скачивание с GitHub/внешних хостов ("unable to get certificate CRL").
    # Сохраняем VERIFY_PEER, но отключаем требование CRL для данного запроса.
    store.flags = 0 if store.respond_to?(:flags=)
    http.verify_mode = OpenSSL::SSL::VERIFY_PEER
    http.cert_store = store
  end
end
