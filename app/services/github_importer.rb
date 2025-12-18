# frozen_string_literal: true

require "net/http"
require "uri"
require "zip"

# Импорт сервиса из GitHub-репозитория: скачивает zip репозитория и отдаёт bytes.
class GithubImporter
  class Error < StandardError; end

  CodeloadHost = "codeload.github.com"

  def self.fetch(repo_url:, ref: nil)
    new(repo_url, ref).fetch
  end

  def initialize(repo_url, ref)
    @repo_url = repo_url.to_s.strip
    @ref = ref.to_s.strip
  end

  def fetch
    owner, repo, parsed_ref = parse_github(@repo_url)
    ref = @ref.presence || parsed_ref.presence || "main"
    zip_bytes = fetch_repo_zip(owner: owner, repo: repo, ref: ref)

    {
      owner: owner.to_s,
      repo: repo.to_s,
      ref: ref.to_s,
      archive_url: github_archive_url(owner: owner, repo: repo, ref: ref),
      zip_bytes: zip_bytes
    }
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

  def github_archive_url(owner:, repo:, ref:)
    # GitHub обычно редиректит на codeload, но ссылка стабильная.
    # Для веток это refs/heads/*, для тегов — refs/tags/*; здесь используем ветку по умолчанию.
    "https://github.com/#{owner}/#{repo}/archive/refs/heads/#{ref}.zip"
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
end
