class Team < ApplicationRecord
  belongs_to :university, optional: true
  belongs_to :captain, class_name: 'User', optional: true

  has_many :team_memberships, dependent: :destroy
  has_many :users, through: :team_memberships
  has_many :membership_events, class_name: 'TeamMembershipEvent', dependent: :destroy
  has_many :writeups, dependent: :destroy

  validates :name, presence: true
  # Глобальное ограничение: один пользователь может быть капитаном только в одной команде
  validates :captain_id, uniqueness: true, allow_nil: true

  validate :validate_avatar_url

  private
  def validate_avatar_url
    url = avatar_url.to_s.strip
    return if url.blank?
    return if url =~ /\A(?:https?:\/\/|data:image)/i
    errors.add(:avatar_url, 'должен начинаться с http(s):// или data:image')
  end
end
