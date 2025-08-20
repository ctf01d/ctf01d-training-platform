class Team < ApplicationRecord
  belongs_to :university, optional: true
  belongs_to :captain, class_name: 'User', optional: true

  has_many :team_memberships, dependent: :destroy
  has_many :users, through: :team_memberships

  validates :name, presence: true
end
