class University < ApplicationRecord
  has_many :teams

  validates :name, presence: true, uniqueness: true
  validate :validate_avatar_url

  private
  def validate_avatar_url
    url = avatar_url.to_s.strip
    return if url.blank?
    return if url =~ /\A(?:https?:\/\/|data:image)/i
    errors.add(:avatar_url, 'должен начинаться с http(s):// или data:image')
  end
end
