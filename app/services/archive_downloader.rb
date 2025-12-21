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

      req = Net::HTTP::Get.new(uri.request_uri)
      FileUtils.mkdir_p(dest_dir)
      fname = filename || safe_filename_from(uri)
      path = File.join(dest_dir, fname)

      tmp = "#{path}.part"
      FileUtils.rm_f(tmp)

      redirect_location = nil
      sha256 = nil
      size = nil

      begin
        http.request(req) do |res|
          if res.is_a?(Net::HTTPRedirection)
            redirect_location = res["location"].to_s
            raise Error, "редирект без Location" if redirect_location.blank?
            next
          end
          unless res.is_a?(Net::HTTPSuccess)
            raise Error, "не удалось скачать: HTTP #{res.code}"
          end

          content_type = res["content-type"]&.split(";")&.first
          first_bytes = +""
          digest = Digest::SHA256.new
          size = 0

          File.open(tmp, "wb") do |f|
            res.read_body do |chunk|
              next if chunk.nil? || chunk.empty?
              size += chunk.bytesize
              raise Error, "слишком большой архив: > #{max_bytes} байт" if size > max_bytes
              if first_bytes.bytesize < 4
                need = 4 - first_bytes.bytesize
                first_bytes << chunk.byteslice(0, need)
              end
              digest.update(chunk)
              f.write(chunk)
            end
          end

          unless zip_magic?(first_bytes)
            FileUtils.rm_f(tmp)
            raise Error, "ожидался zip-архив, получен #{content_type || 'unknown'}"
          end

          sha256 = digest.hexdigest
        end
      ensure
        if redirect_location.present?
          FileUtils.rm_f(tmp)
        end
      end

      if redirect_location.present?
        uri = URI.join(uri.to_s, redirect_location)
        redirects += 1
        next
      end

      FileUtils.mv(tmp, path)
      return { path: path, size: size || File.size(path), sha256: sha256 || Digest::SHA256.file(path).hexdigest, content_type: content_type }
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

    if uploaded_file.respond_to?(:size) && uploaded_file.size.to_i > max_bytes
      raise Error, "слишком большой архив: > #{max_bytes} байт"
    end

    content_type = (uploaded_file.respond_to?(:content_type) && uploaded_file.content_type).to_s.split(";").first

    FileUtils.mkdir_p(dest_dir)
    fname = filename || sanitize_filename(uploaded_file.respond_to?(:original_filename) ? uploaded_file.original_filename : "archive.zip")
    fname = ensure_zip_ext(fname)
    path = File.join(dest_dir, fname)

    tmp = "#{path}.part"
    FileUtils.rm_f(tmp)
    first_bytes = +""
    digest = Digest::SHA256.new
    size = 0

    File.open(tmp, "wb") do |f|
      io.rewind if io.respond_to?(:rewind)
      while (chunk = io.read(16 * 1024))
        next if chunk.empty?
        size += chunk.bytesize
        raise Error, "слишком большой архив: > #{max_bytes} байт" if size > max_bytes
        if first_bytes.bytesize < 4
          need = 4 - first_bytes.bytesize
          first_bytes << chunk.byteslice(0, need)
        end
        digest.update(chunk)
        f.write(chunk)
      end
    end

    unless zip_magic?(first_bytes)
      FileUtils.rm_f(tmp)
      raise Error, "ожидался zip-архив, получен #{content_type.presence || 'unknown'}"
    end

    FileUtils.mv(tmp, path)
    { path: path, size: size, sha256: digest.hexdigest, content_type: content_type }
  end

  private
  def zip_magic?(bytes)
    return false unless bytes && bytes.bytesize >= 2
    bytes.bytes[0] == 0x50 && bytes.bytes[1] == 0x4B
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
