class Game < ApplicationRecord
  has_many :results, dependent: :destroy
  has_many :teams, through: :results
  has_and_belongs_to_many :services
  has_many :final_results, dependent: :destroy
  has_many :writeups, dependent: :destroy

  validates :name, presence: true
  validate :validate_avatar_url
  validate :validate_urls

  scope :upcoming, -> { where("starts_at > ?", Time.current) }
  scope :ongoing,  -> { where("starts_at <= ? AND ends_at >= ?", Time.current, Time.current) }
  scope :past,     -> { where("ends_at < ?", Time.current) }

  def status(now = Time.current)
    return :ongoing if starts_at.present? && ends_at.present? && starts_at <= now && ends_at >= now
    return :upcoming if starts_at.present? && starts_at > now
    return :past if ends_at.present? && ends_at < now
    :unknown
  end

  def registration_status(now = Time.current)
    return :unscheduled if registration_opens_at.blank? && registration_closes_at.blank?
    open = registration_opens_at
    close = registration_closes_at
    return :upcoming if open.present? && now < open
    return :open if (open.blank? || now >= open) && (close.blank? || now <= close)
    :closed
  end

  def scoreboard_status(now = Time.current)
    # если окно не задано — считаем всегда открытым
    return :always if scoreboard_opens_at.blank? && scoreboard_closes_at.blank?
    open = scoreboard_opens_at
    close = scoreboard_closes_at
    return :upcoming if open.present? && now < open
    return :open if (open.blank? || now >= open) && (close.blank? || now <= close)
    :closed
  end

  private
  def validate_avatar_url
    url = avatar_url.to_s.strip
    return if url.blank?
    return if url =~ /\A(?:https?:\/\/|data:image)/i
    errors.add(:avatar_url, "должен начинаться с http(s):// или data:image")
  end

  def validate_urls
    { vpn_url: vpn_url, vpn_config_url: vpn_config_url }.each do |field, value|
      v = value.to_s.strip
      next if v.blank?
      next if v =~ /\Ahttps?:\/\//i
      errors.add(field, "должен начинаться с http(s)://")
    end
  end
end
