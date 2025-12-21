# frozen_string_literal: true

require "zip"

# Проверка наличия чекера в bundle.zip (service/ + checker/) и простая "валидация" по маркерам 101..104.
class CheckerInspector
  class Error < StandardError; end

  REQUIRED_CODES = %w[101 102 103 104].freeze
  MAX_ENTRY_BYTES = 2 * 1024 * 1024

  def self.call(zip_path:)
    new(zip_path).call
  end

  def initialize(zip_path)
    @zip_path = zip_path.to_s
  end

  def call
    raise Error, "архив не найден: #{@zip_path}" unless File.file?(@zip_path)

    has_checker = false
    found = {}

    Zip::File.open(@zip_path) do |zip|
      zip.each do |entry|
        name = entry.name.to_s
        next if name.blank?
        next unless name =~ %r{(^|/)checker/}
        has_checker = true
        next if entry.directory?

        data = entry.get_input_stream.read(MAX_ENTRY_BYTES).to_s
        REQUIRED_CODES.each do |code|
          found[code] ||= contains_token?(data, code)
        end
        break if found.size == REQUIRED_CODES.size
      end
    end

    return { status: "missing", found_codes: [] } unless has_checker
    return { status: "codes", found_codes: REQUIRED_CODES } if found.size == REQUIRED_CODES.size
    { status: "present", found_codes: found.keys.sort }
  rescue Zip::Error => e
    raise Error, "не удалось прочитать zip: #{e.message}"
  end

  private
  def contains_token?(data, token)
    return false if data.blank? || token.blank?
    i = 0
    while (pos = data.index(token, i))
      before = pos.zero? ? nil : data.getbyte(pos - 1)
      after = data.getbyte(pos + token.bytesize)
      before_ok = before.nil? || before < 48 || before > 57
      after_ok = after.nil? || after < 48 || after > 57
      return true if before_ok && after_ok
      i = pos + 1
    end
    false
  end
end
