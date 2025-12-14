class GameTeam < ApplicationRecord
  belongs_to :game
  belongs_to :team

  attr_accessor :ctf01d_overrides_text

  scope :ordered, -> { order(:order, :id) }

  validates :game_id, presence: true
  validates :team_id, presence: true, uniqueness: { scope: :game_id }
  validates :order, numericality: { only_integer: true }
  validates :ip_address, format: { with: /\A(?:\d{1,3}\.){3}\d{1,3}\z/, allow_blank: true }
  validates :ctf01d_id, uniqueness: { scope: :game_id, allow_blank: true }
  validate  :validate_overrides_keys

  before_validation :normalize_overrides_text
  before_validation :normalize_overrides_hash
  before_validation :normalize_ctf01d_id
  before_validation :normalize_ip_address

  def ctf01d_overrides_text
    return @ctf01d_overrides_text unless @ctf01d_overrides_text.nil?
    (ctf01d_overrides || {}).map { |k, v| "#{k}: #{v}" }.join("\n")
  end

  def ctf01d_extra_hash
    (ctf01d_overrides || {}).each_with_object({}) do |(k, v), acc|
      next if v.nil? || v.to_s.strip.empty?
      acc[k.to_s] = v
    end
  end

  private

  def normalize_overrides_text
    return if @ctf01d_overrides_text.nil?
    parsed = parse_lines(@ctf01d_overrides_text)
    self.ctf01d_id = parsed.delete("ctf01d_id").presence || ctf01d_id
    self.ctf01d_overrides = parsed
  end

  def normalize_overrides_hash
    self.ctf01d_overrides ||= {}
    self.ctf01d_overrides = ctf01d_overrides.to_h.stringify_keys
  end

  def normalize_ctf01d_id
    return if ctf01d_id.blank?
    self.ctf01d_id = ctf01d_id.to_s.strip.downcase.gsub(/[^a-z0-9]+/, "")
  end

  def normalize_ip_address
    self.ip_address = ip_address.to_s.strip.presence
  end

  def parse_lines(text)
    text.to_s.lines.each_with_object({}) do |raw, acc|
      line = raw.strip
      next if line.empty?
      key, value = line.split(":", 2)
      key = key.to_s.strip
      next if key.empty?
      acc[key] = value.to_s.strip
    end
  end

  def validate_overrides_keys
    bad_keys = (ctf01d_overrides || {}).keys.reject { |k| k.to_s.start_with?("ctf01d_") }
    return if bad_keys.empty?
    errors.add(:ctf01d_overrides, "ключи должны начинаться с ctf01d_ (#{bad_keys.join(', ')})")
  end
end
