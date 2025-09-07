# frozen_string_literal: true

require "net/http"
require "uri"
require "zip"
require "fileutils"
require "tempfile"

# Импорт сервиса из GitHub-репозитория: формирует отдельные zip для service/ и checker/
class GithubImporter
  class Error < StandardError; end

  CodeloadHost = "codeload.github.com"

  def self.import(repo_url:, ref: nil, unpack_to: nil, folder_name: nil)
    new(repo_url, ref).import(unpack_to: unpack_to, folder_name: folder_name)
  end

  def initialize(repo_url, ref)
    @repo_url = repo_url.to_s.strip
    @ref = ref.to_s.strip
  end

  def import(unpack_to: nil, folder_name: nil)
    owner, repo, parsed_ref = parse_github(@repo_url)
    ref = @ref.presence || parsed_ref.presence || "main"
    zip_bytes = fetch_repo_zip(owner: owner, repo: repo, ref: ref)

    # Выделим корневую папку архива (<repo>-<ref>/)
    root_prefix = detect_root_prefix(zip_bytes)

    service_zip = build_subdir_zip(zip_bytes, File.join(root_prefix, "service/"))
    checker_zip = build_subdir_zip(zip_bytes, File.join(root_prefix, "checker/"))

    # Попробуем извлечь имя из первого заголовка README; автор — из пути (owner)
    repo_slug = repo.to_s
    owner_slug = owner.to_s
    name = repo_slug.tr("-", " ").strip
    begin
      readme = read_entry(zip_bytes, File.join(root_prefix, "service/README.md")) ||
               read_entry(zip_bytes, File.join(root_prefix, "README.md")) ||
               read_entry(zip_bytes, File.join(root_prefix, "readme.md"))
    rescue StandardError
      readme = nil
    end
    if readme.present?
      title = extract_title(readme)
      name = title if title.present?
    end
    public_desc = summarize_markdown(readme) if readme

    # LICENSE → вычислим копирайт и тип лицензии (распознаём топ-10 SPDX)
    license_text = read_license(zip_bytes, root_prefix: root_prefix)
    cr = extract_copyright(license_text) if license_text.present?
    lic = detect_license(license_text) if license_text.present?

    result = {
      name: name.presence || repo_slug,
      author: owner_slug,
      public_description: public_desc,
      copyright: cr,
      license: lic,
      archives: {
        service: service_zip, # StringIO bytes
        checker: checker_zip
      }
    }

    if unpack_to.present?
      dest_base = File.expand_path(unpack_to.to_s)
      repo_dir_name = (folder_name.presence || repo).to_s
      dest_dir = File.join(dest_base, repo_dir_name)
      unpack_repo(zip_bytes, root_prefix: root_prefix, dest_dir: dest_dir)
      result[:unpacked_path] = dest_dir
    end

    result
  end

  private
  def parse_github(url)
    uri = URI.parse(url)
    raise Error, "невалидный URL" unless uri.host&.end_with?("github.com")
    parts = uri.path.to_s.split("/").reject(&:blank?)
    raise Error, "некорректный путь, ожидается /owner/repo" unless parts.size >= 2
    owner = parts[0]
    repo = parts[1].sub(/\.git\z/i, "")
    ref = nil
    if parts[2] == "tree" && parts[3].present?
      ref = parts[3]
    end
    [ owner, repo, ref ]
  end

  def fetch_repo_zip(owner:, repo:, ref:)
    # Порядок попыток: heads -> tags
    [ "refs/heads/#{ref}", "refs/tags/#{ref}", ref ].each do |ref_path|
      path = "/#{owner}/#{repo}/zip/#{ref_path}"
      bytes = http_get(CodeloadHost, path)
      return bytes if bytes
    end
    raise Error, "не удалось скачать архив репозитория #{owner}/#{repo}@#{ref}"
  end

  def http_get(host, path)
    http = Net::HTTP.new(host, 443)
    http.use_ssl = true
    http.open_timeout = 5
    http.read_timeout = 60
    res = http.get(path)
    return nil unless res.is_a?(Net::HTTPSuccess)
    res.body
  rescue StandardError
    nil
  end

  def detect_root_prefix(zip_bytes)
    buffer = StringIO.new(zip_bytes)
    root = nil
    Zip::File.open_buffer(buffer) do |zip|
      root = zip.first.name.split("/").first + "/"
    end
    root || ""
  end

  def read_entry(zip_bytes, entry_name)
    buffer = StringIO.new(zip_bytes)
    Zip::File.open_buffer(buffer) do |zip|
      entry = zip.find_entry(entry_name)
      return entry.get_input_stream.read if entry
    end
    nil
  end

  def read_license(zip_bytes, root_prefix:)
    candidates = [
      "LICENSE", "LICENSE.txt", "LICENSE.md", "LICENCE", "LICENCE.txt", "COPYING", "COPYING.txt"
    ]
    # Сначала в корне репозитория, затем внутри service/
    paths = candidates.map { |n| File.join(root_prefix, n) } +
            candidates.map { |n| File.join(root_prefix, "service", n) }
    buffer = StringIO.new(zip_bytes)
    Zip::File.open_buffer(buffer) do |zip|
      paths.each do |p|
        entry = zip.find_entry(p)
        return entry.get_input_stream.read if entry
      end
    end
    nil
  end

  def build_subdir_zip(zip_bytes, subdir_prefix)
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

  def unpack_repo(zip_bytes, root_prefix:, dest_dir:)
    # Разворачиваем весь репозиторий в dest_dir, обрезая root_prefix из путей
    if File.exist?(dest_dir)
      # Разрешаем пустую директорию, иначе — ошибка, чтобы не затирать данные
      raise Error, "папка уже существует и не пуста: #{dest_dir}" unless Dir.exist?(dest_dir) && (Dir.children(dest_dir).empty?)
    else
      FileUtils.mkdir_p(dest_dir)
    end

    Zip::File.open_buffer(StringIO.new(zip_bytes)) do |zip|
      zip.each do |entry|
        next unless entry.name.start_with?(root_prefix)
        rel = entry.name.sub(root_prefix, "")
        next if rel.empty?
        dst = File.join(dest_dir, rel)
        if entry.directory?
          FileUtils.mkdir_p(dst)
        else
          FileUtils.mkdir_p(File.dirname(dst))
          entry.extract(dst) { true }
        end
      end
    end
  end

  def summarize_markdown(md)
    txt = md.to_s
    # простая выжимка: первые 400 символов без заголовков/ссылок
    txt = txt.gsub(/\[(.*?)\]\((.*?)\)/, '\\1')
    txt = txt.gsub(/^\s*#.*$/m, "")
    txt.strip[0, 400]
  end

  def extract_title(md)
    return nil if md.to_s.strip.empty?
    lines = md.to_s.lines
    # ищем заголовок уровня 1, затем 2
    line = lines.find { |l| l.strip.start_with?("# ") } || lines.find { |l| l.strip.start_with?("## ") }
    return nil unless line
    title = line.sub(/^#+\s*/, "")
    # убрать markdown-ссылки и инлайн-код
    title = title.gsub(/\[(.*?)\]\((.*?)\)/, '\\1').gsub(/`+([^`]+)`+/, '\\1')
    title.strip
  end

  def extract_copyright(license_text)
    return nil if license_text.to_s.strip.empty?
    line = license_text.to_s.each_line.find { |l| l =~ /copyright/i }
    return nil unless line
    v = line.strip
    # Уберём префиксы вроде (c), © и лишние пробелы
    v = v.sub(/^\s*\(c\)\s*/i, "").sub(/^\s*©\s*/, "")
    v = v.gsub(/\s+/, " ").strip
    # Ограничим длину для хранения в string
    v[0, 200]
  end

  def detect_license(text)
    t = text.to_s.downcase
    return nil if t.strip.empty?

    # MIT
    if t.include?("mit license") || t.include?("permission is hereby granted, free of charge")
      return "MIT"
    end

    # Apache-2.0
    if (t.include?("apache license") && t.include?("version 2.0")) || t.include?("apache-2.0")
      return "Apache-2.0"
    end

    # BSD 3-Clause / 2-Clause (грубая эвристика)
    if t.include?("redistribution and use in source and binary forms")
      if t.include?("neither the name of the") || t.include?("neither the name nor the names")
        return "BSD-3-Clause"
      else
        return "BSD-2-Clause"
      end
    end

    # GPL-3.0 / GPL-2.0
    if t.include?("gnu general public license")
      if t.include?("version 3") || t.include?("gpl version 3") || t.include?("gplv3")
        return "GPL-3.0"
      elsif t.include?("version 2") || t.include?("gpl version 2") || t.include?("gplv2")
        return "GPL-2.0"
      else
        return "GPL"
      end
    end

    # LGPL-3.0
    if t.include?("gnu lesser general public license") || t.include?("lgpl")
      if t.include?("version 3") || t.include?("lgplv3")
        return "LGPL-3.0"
      else
        return "LGPL"
      end
    end

    # MPL-2.0
    if t.include?("mozilla public license")
      return "MPL-2.0"
    end

    # ISC
    if t.include?("isc license") || (t.include?("permission to use, copy, modify") && t.include?("the author and contributors"))
      return "ISC"
    end

    # Unlicense
    if t.include?("this is free and unencumbered software released into the public domain") || t.include?("unlicense")
      return "Unlicense"
    end

    nil
  end
end
