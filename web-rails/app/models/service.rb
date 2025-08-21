class Service < ApplicationRecord
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
        next if val =~ %r{\A(data:image|https?://)}i
      else
        next if val =~ %r{\Ahttps?://}i
      end
      errors.add(field, 'должен начинаться с http(s)://')
    end
  end
end
