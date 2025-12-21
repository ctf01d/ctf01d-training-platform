class Service < ApplicationRecord
  require "uri"

  has_and_belongs_to_many :games

  URL_FIELDS = %i[
    avatar_url service_archive_url checker_archive_url writeup_url exploits_url
  ].freeze

  validates :name, presence: true, uniqueness: true
  validate :validate_urls

  scope :publicly_visible, -> { where(public: true) }

  private
  def validate_urls
    URL_FIELDS.each do |field|
      val = self.send(field).to_s.strip
      next if val.blank?
      if field == :avatar_url
        next if val =~ %r{\Adata:image}i
        next if valid_http_url?(val)
      else
        next if valid_http_url?(val)
      end
      errors.add(field, "должен быть валидным http(s):// URL")
    end
  end

  def valid_http_url?(value)
    uri = URI.parse(value.to_s)
    return false unless uri.is_a?(URI::HTTP)
    return false if uri.host.blank?
    %w[http https].include?(uri.scheme)
  rescue URI::InvalidURIError
    false
  end
end
