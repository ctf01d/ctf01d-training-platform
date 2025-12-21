module ApplicationHelper
  require "uri"

  def safe_http_url(url)
    v = url.to_s.strip
    return nil if v.blank?
    uri = URI.parse(v)
    return nil unless uri.is_a?(URI::HTTP) && uri.host.present?
    return nil unless %w[http https].include?(uri.scheme)
    v
  rescue URI::InvalidURIError
    nil
  end
end
