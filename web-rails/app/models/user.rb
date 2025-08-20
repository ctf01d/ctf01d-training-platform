class User < ApplicationRecord
  has_secure_password

  has_many :team_memberships, dependent: :destroy
  has_many :teams, through: :team_memberships
  has_many :membership_events, class_name: 'TeamMembershipEvent', dependent: :nullify
  has_many :authored_membership_events, class_name: 'TeamMembershipEvent', foreign_key: 'actor_id', dependent: :nullify

  validates :user_name, presence: true, uniqueness: true,
                        format: { with: /\A[a-zA-Z0-9_]+\z/, message: 'латиница, цифры и _' }
  validates :display_name, presence: true
  validates :role, presence: true
end
