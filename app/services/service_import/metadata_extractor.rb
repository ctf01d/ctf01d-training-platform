# frozen_string_literal: true

require "zip"

module ServiceImport
  # Сервис: извлечь метаданные (name/description/license) из bundle.zip (service/ + checker/).
  class MetadataExtractor
    MAX_TEXT_BYTES = 512 * 1024

    def self.call(bundle_zip_bytes:)
      new(bundle_zip_bytes).call
    end

    def initialize(bundle_zip_bytes)
      @bundle_zip_bytes = bundle_zip_bytes
    end

    def call
      readme = read_entry(@bundle_zip_bytes, "service/README.md") ||
               read_entry(@bundle_zip_bytes, "service/readme.md") ||
               read_entry(@bundle_zip_bytes, "service/README") ||
               read_entry(@bundle_zip_bytes, "service/readme")

      license_text = read_license(@bundle_zip_bytes)

      title = extract_title(readme) if readme.present?
      public_desc = summarize_markdown(readme) if readme.present?

      lic = detect_license(license_text) if license_text.present?
      cr = extract_copyright(license_text) if license_text.present?

      {
        name: title,
        public_description: public_desc,
        copyright: cr,
        license: lic
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

    def read_small_entry(entry)
      return nil unless entry
      data = +""
      entry.get_input_stream do |io|
        while (chunk = io.read(16 * 1024))
          break if chunk.empty?
          data << chunk
          break if data.bytesize >= MAX_TEXT_BYTES
        end
      end
      data.byteslice(0, MAX_TEXT_BYTES)
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
