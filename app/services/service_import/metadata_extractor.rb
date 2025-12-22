# frozen_string_literal: true

require "zip"
require "json"

module ServiceImport
  # Сервис: извлечь метаданные (name/description/license) из bundle.zip (service/ + checker/).
  class MetadataExtractor
    MAX_TEXT_BYTES = 512 * 1024
    TRAINING_JSON_MAX_BYTES = 512 * 1024

    def self.call(bundle_zip_bytes:)
      new(bundle_zip_bytes).call
    end

    def initialize(bundle_zip_bytes)
      @bundle_zip_bytes = normalize_bytes(bundle_zip_bytes)
    end

    def call
      readme = read_entry(@bundle_zip_bytes, "service/README.md") ||
               read_entry(@bundle_zip_bytes, "service/readme.md") ||
               read_entry(@bundle_zip_bytes, "service/README") ||
               read_entry(@bundle_zip_bytes, "service/readme")

      license_text = read_license(@bundle_zip_bytes)
      training_json = read_training_json(@bundle_zip_bytes)
      training = parse_training_json(training_json) if training_json.present?

      title = training&.dig("display_name").to_s.strip.presence || (extract_title(readme) if readme.present?)
      public_desc = training&.dig("description").to_s.strip.presence || (summarize_markdown(readme) if readme.present?)

      lic = detect_license(license_text) if license_text.present?
      cr = extract_copyright(license_text) if license_text.present?

      {
        name: title,
        public_description: public_desc,
        copyright: cr,
        license: lic,
        ctf01d_training: training
      }
    end

    private
    def read_entry(zip_bytes, entry_name)
      Zip::File.open_buffer(StringIO.new(zip_bytes)) do |zip|
        entry = zip.find_entry(entry_name)
        return read_small_entry(entry) if entry
      end
      nil
    rescue Zip::Error
      nil
    end

    def read_license(zip_bytes)
      candidates = [
        "service/LICENSE", "service/LICENSE.txt", "service/LICENSE.md",
        "service/LICENCE", "service/LICENCE.txt",
        "service/COPYING", "service/COPYING.txt"
      ]
      Zip::File.open_buffer(StringIO.new(zip_bytes)) do |zip|
        candidates.each do |p|
          entry = zip.find_entry(p)
          return read_small_entry(entry) if entry
        end
      end
      nil
    rescue Zip::Error
      nil
    end

    def read_training_json(zip_bytes)
      Zip::File.open_buffer(StringIO.new(zip_bytes)) do |zip|
        entry = zip.find_entry("service/ctf01d-training.json")
        return read_small_entry(entry, max_bytes: TRAINING_JSON_MAX_BYTES) if entry

        fallback = zip.find_all do |e|
          name = e.name.to_s
          name.end_with?("ctf01d-training.json") && !name.start_with?("checker/")
        end.min_by { |e| e.name.to_s.length }
        return read_small_entry(fallback, max_bytes: TRAINING_JSON_MAX_BYTES) if fallback
      end
      nil
    rescue Zip::Error
      nil
    end

    def parse_training_json(text)
      obj = JSON.parse(text.to_s)
      return nil unless obj.is_a?(Hash)
      obj
    rescue JSON::ParserError
      nil
    end

    def normalize_bytes(value)
      return value if value.is_a?(String)
      return "" if value.nil?

      if value.respond_to?(:read)
        data = value.read.to_s
        value.rewind if value.respond_to?(:rewind)
        return data
      end

      value.to_s
    end

    def read_small_entry(entry, max_bytes: MAX_TEXT_BYTES)
      return nil unless entry
      data = +""
      entry.get_input_stream do |io|
        while (chunk = io.read(16 * 1024))
          break if chunk.empty?
          data << chunk
          break if data.bytesize >= max_bytes
        end
      end
      data.byteslice(0, max_bytes)
    end

    def summarize_markdown(md)
      txt = md.to_s
      txt = txt.gsub(/\[(.*?)\]\((.*?)\)/, '\\1')
      txt = txt.lines.reject { |l| l.strip.start_with?("#") }.join
      txt.strip[0, 400]
    end

    def extract_title(md)
      return nil if md.to_s.strip.empty?
      lines = md.to_s.lines
      line = lines.find { |l| l.strip.start_with?("# ") } || lines.find { |l| l.strip.start_with?("## ") }
      return nil unless line
      title = line.sub(/^#+\s*/, "")
      title = title.gsub(/\[(.*?)\]\((.*?)\)/, '\\1').gsub(/`+([^`]+)`+/, '\\1')
      title.strip
    end

    def extract_copyright(license_text)
      return nil if license_text.to_s.strip.empty?
      line = license_text.to_s.each_line.find { |l| l =~ /copyright/i }
      return nil unless line
      v = line.strip
      v = v.sub(/^\s*\(c\)\s*/i, "").sub(/^\s*©\s*/, "")
      v = v.gsub(/\s+/, " ").strip
      v[0, 200]
    end

    def detect_license(text)
      t = text.to_s.downcase
      return nil if t.strip.empty?

      if t.include?("mit license") || t.include?("permission is hereby granted, free of charge")
        return "MIT"
      end

      if (t.include?("apache license") && t.include?("version 2.0")) || t.include?("apache-2.0")
        return "Apache-2.0"
      end

      if t.include?("redistribution and use in source and binary forms")
        if t.include?("neither the name of the") || t.include?("neither the name nor the names")
          return "BSD-3-Clause"
        else
          return "BSD-2-Clause"
        end
      end

      if t.include?("gnu general public license")
        if t.include?("version 3") || t.include?("gpl version 3") || t.include?("gplv3")
          return "GPL-3.0"
        elsif t.include?("version 2") || t.include?("gpl version 2") || t.include?("gplv2")
          return "GPL-2.0"
        else
          return "GPL"
        end
      end

      if t.include?("gnu lesser general public license") || t.include?("lgpl")
        if t.include?("version 3") || t.include?("lgplv3")
          return "LGPL-3.0"
        else
          return "LGPL"
        end
      end

      if t.include?("mozilla public license")
        return "MPL-2.0"
      end

      if t.include?("isc license") || (t.include?("permission to use, copy, modify") && t.include?("the author and contributors"))
        return "ISC"
      end

      if t.include?("this is free and unencumbered software released into the public domain") || t.include?("unlicense")
        return "Unlicense"
      end

      nil
    end
  end
end
