class TeamMembershipEvent < ApplicationRecord
  belongs_to :team
  belongs_to :user
  belongs_to :actor, class_name: 'User', optional: true

  ACTIONS = %w[created invited join_requested approved rejected accepted declined role_changed removed left].freeze

  validates :action, presence: true, inclusion: { in: ACTIONS }
end

