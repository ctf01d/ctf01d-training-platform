class Team < ApplicationRecord
  belongs_to :university, optional: true
  belongs_to :captain, class_name: 'User', optional: true

  has_many :team_memberships, dependent: :destroy
  has_many :users, through: :team_memberships
  has_many :membership_events, class_name: 'TeamMembershipEvent', dependent: :destroy

  validates :name, presence: true
  # Глобальное ограничение: один пользователь может быть капитаном только в одной команде
  validates :captain_id, uniqueness: true, allow_nil: true
end
