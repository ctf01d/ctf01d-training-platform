class Writeup < ApplicationRecord
  require "uri"

  belongs_to :game
  belongs_to :team

  validates :title, presence: true, length: { maximum: 255 }
  validates :url, presence: true
  validate :validate_url
  validates :team_id, uniqueness: { scope: [ :game_id, :title ], message: "уже есть writeup с таким названием для этой игры" }

  private
  def validate_url
    v = url.to_s.strip
    return if v.blank?
    uri = URI.parse(v)
    return if uri.is_a?(URI::HTTP) && uri.host.present? && %w[http https].include?(uri.scheme)
    errors.add(:url, "должен быть валидным http(s):// URL")
  rescue URI::InvalidURIError
    errors.add(:url, "должен быть валидным http(s):// URL")
  end
end
