#!/usr/bin/env ruby
require "json"
require "net/http"
require "uri"
require "fileutils"
require "openssl"

API = URI("https://ru.wikipedia.org/w/api.php") # при необходимости смените на en.wikipedia.org
INPUT = File.expand_path("universities.txt", __dir__)
OUT_DIR = File.expand_path("downloaded-logos", __dir__)
FileUtils.mkdir_p(OUT_DIR)

def safe_filename(text)
  t = text.to_s.strip
  t = "logo" if t.empty?
  t = t.gsub(/[<>:"'`\/\\|?*\n\r]/, "_") # запрещённые символы
  t = t.gsub(/\s+/, "_")
  t = "logo" if t.empty?
  t
end

def http_get(uri)
  Net::HTTP.start(uri.host, uri.port, use_ssl: uri.scheme == "https", verify_mode: OpenSSL::SSL::VERIFY_NONE) do |http|
    req = Net::HTTP::Get.new(uri)
    res = http.request(req)
    raise "HTTP #{res.code}" unless res.is_a?(Net::HTTPSuccess)
    res
  end
end

def fetch_logo(name)
  # 1) поиск страницы
  search_params = {
    action: "query", list: "search", srsearch: name, format: "json", srlimit: 1
  }
  search_uri = API.dup
  search_uri.query = URI.encode_www_form(search_params)
  data = JSON.parse(http_get(search_uri).body)
  hit = data.dig("query", "search")&.first
  unless hit
    puts "[skip] #{name} — нет результатов"
    return
  end
  title = hit["title"]

  # 2) пробуем найти картинку-логотип в списке файлов статьи
  images_params = {
    action: "query", prop: "images", format: "json", titles: title, imlimit: 50
  }
  images_uri = API.dup
  images_uri.query = URI.encode_www_form(images_params)
  images_pages = JSON.parse(http_get(images_uri).body).dig("query", "pages") || {}
  image_titles = images_pages.values.flat_map { |p| p["images"] || [] }.map { |img| img["title"] }
  exclude_regex = /wikimedia|commons|wikipedia|герб_россии|coat_of_arms_of_the_russian_federation/i
  logo_regex = /logo|logotype|логотип|эмблем|emblem/i
  svg_titles = image_titles.select { |t| t =~ /\.svg$/i && t !~ exclude_regex }
  filtered = image_titles.reject { |t| t =~ exclude_regex }
  logo_titles = filtered.select { |t| t =~ logo_regex }
  emblem_titles = filtered.select { |t| t =~ /герб|coat_of_arms/i }

  logo_url = nil
  chosen = logo_titles.first || svg_titles.first || emblem_titles.first || filtered.first

  if chosen
    info_params = {
      action: "query", prop: "imageinfo", format: "json",
      iiprop: "url", titles: chosen
    }
    info_uri = API.dup
    info_uri.query = URI.encode_www_form(info_params)
    info_pages = JSON.parse(http_get(info_uri).body).dig("query", "pages") || {}
    logo_url = info_pages.values.map { |p| p.dig("imageinfo", 0, "url") }.compact.first
  end

  # 3) fallback — thumbnail первой картинки (pageimage)
  if logo_url.nil?
    thumb_params = {
      action: "query", prop: "pageimages", format: "json",
      piprop: "thumbnail", pithumbsize: 512, titles: title
    }
    thumb_uri = API.dup
    thumb_uri.query = URI.encode_www_form(thumb_params)
    pages = JSON.parse(http_get(thumb_uri).body).dig("query", "pages") || {}
    logo_url = pages.values.map { |p| p.dig("thumbnail", "source") }.compact.first
  end

  unless logo_url
    puts "[skip] #{name} — нет thumbnail"
    return
  end

  # 4) скачиваем с именем на базе исходного названия
  ext = File.extname(URI.parse(logo_url).path)
  ext = ".jpg" if ext.empty?
  fname = File.join(OUT_DIR, "#{safe_filename(name)}#{ext}")
  img_uri = URI(logo_url)
  File.binwrite(fname, http_get(img_uri).body)
  puts "[ok] #{name} -> #{fname}"
end

unless File.exist?(INPUT)
  warn "Нет файла #{INPUT}. Положите туда названия (по строке)."
  exit 1
end

File.read(INPUT).lines.map(&:strip).reject(&:empty?).each do |name|
  fetch_logo(name)
rescue => e
  warn "[error] #{name}: #{e.message}"
end
