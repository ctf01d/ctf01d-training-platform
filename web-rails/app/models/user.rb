class User < ApplicationRecord
  has_secure_password

  has_many :team_memberships, dependent: :destroy
  has_many :teams, through: :team_memberships

  validates :user_name, presence: true, uniqueness: true,
                        format: { with: /\A[a-zA-Z0-9_]+\z/, message: 'латиница, цифры и _' }
  validates :display_name, presence: true
  validates :role, presence: true
end
