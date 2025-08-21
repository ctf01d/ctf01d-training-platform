class Game < ApplicationRecord
  has_many :results, dependent: :destroy
  has_many :teams, through: :results
  has_and_belongs_to_many :services
  has_many :final_results, dependent: :destroy
  has_many :writeups, dependent: :destroy

  validates :name, presence: true

  scope :upcoming, -> { where("starts_at > ?", Time.current) }
  scope :ongoing,  -> { where("starts_at <= ? AND ends_at >= ?", Time.current, Time.current) }
  scope :past,     -> { where("ends_at < ?", Time.current) }

  def status(now = Time.current)
    return :ongoing if starts_at.present? && ends_at.present? && starts_at <= now && ends_at >= now
    return :upcoming if starts_at.present? && starts_at > now
    return :past if ends_at.present? && ends_at < now
    :unknown
  end
end
