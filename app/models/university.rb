class University < ApplicationRecord
  require "uri"

  has_many :teams

  validates :name, presence: true, uniqueness: true
  validate :validate_avatar_url
  validate :validate_site_url

  private
  def validate_avatar_url
    url = avatar_url.to_s.strip
    return if url.blank?
    return if url =~ /\A(?:https?:\/\/|data:image)/i
    errors.add(:avatar_url, "должен начинаться с http(s):// или data:image")
  end

  def validate_site_url
    v = site_url.to_s.strip
    return if v.blank?
    uri = URI.parse(v)
    return if uri.is_a?(URI::HTTP) && uri.host.present? && %w[http https].include?(uri.scheme)
    errors.add(:site_url, "должен быть валидным http(s):// URL")
  rescue URI::InvalidURIError
    errors.add(:site_url, "должен быть валидным http(s):// URL")
  end
end
