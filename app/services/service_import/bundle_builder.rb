# frozen_string_literal: true

require "zip"

module ServiceImport
  # Сервис: собрать единый bundle.zip (service/ + опционально checker/) из произвольного zip.
  # Дополнительно подтягивает README/LICENSE из корня в service/, если внутри service/ их нет.
  class BundleBuilder
    class Error < StandardError; end
    MAX_ENTRY_BYTES = 50 * 1024 * 1024
    MAX_TOTAL_BYTES = 200 * 1024 * 1024
    MAX_FILES = 10_000

    def self.call(zip_bytes:)
      new(zip_bytes).call
    end

    def initialize(zip_bytes)
      @zip_bytes = zip_bytes
      @total_bytes = 0
      @files = 0
    end

    def call
      raise Error, "пустой zip" if @zip_bytes.blank?

      root_prefix = detect_root_prefix(@zip_bytes)

      service_prefix = File.join(root_prefix, "service/")
      checker_prefix = File.join(root_prefix, "checker/")

      buffer = StringIO.new
      service_found = false

      root_readme = nil
      root_readme_name = nil
      root_license = nil
      root_license_name = nil
      service_has_readme = false
      service_has_license = false

      Zip::OutputStream.write_buffer(buffer) do |zos|
        Zip::File.open_buffer(StringIO.new(@zip_bytes)) do |zip|
          has_service = zip.any? { |e| e.name.start_with?(service_prefix) }
          has_checker = zip.any? { |e| e.name.start_with?(checker_prefix) }

          root_readme_name, root_readme = read_first(zip, root_prefix, readme_candidates)
          root_license_name, root_license = read_first(zip, root_prefix, license_candidates)

          if has_service
            service_found = copy_tree(zip, zos, service_prefix, "service/") do |rel|
              service_has_readme ||= readme_candidates.any? { |n| rel.casecmp?(n) }
              service_has_license ||= license_candidates.any? { |n| rel.casecmp?(n) }
            end
          else
            excludes = has_checker ? [ "checker/" ] : []
            service_found = copy_tree(zip, zos, root_prefix, "service/", exclude_rel_prefixes: excludes) do |rel|
              service_has_readme ||= readme_candidates.any? { |n| rel.casecmp?(n) }
              service_has_license ||= license_candidates.any? { |n| rel.casecmp?(n) }
            end
          end

          copy_tree(zip, zos, checker_prefix, "checker/") if has_checker

          if root_readme.present? && !service_has_readme
            dest = "service/#{File.basename(root_readme_name)}"
            zos.put_next_entry(dest)
            zos.write(root_readme)
          end

          if root_license.present? && !service_has_license
            dest = "service/#{File.basename(root_license_name)}"
            zos.put_next_entry(dest)
            zos.write(root_license)
          end
        end
      end

      buffer.rewind
      bytes = buffer.read
      raise Error, "в архиве не найдено содержимое для service/" unless service_found
      bytes
    rescue Zip::Error => e
      raise Error, "не удалось прочитать zip: #{e.message}"
    end

    private
    def detect_root_prefix(zip_bytes)
      buffer = StringIO.new(zip_bytes)
      Zip::File.open_buffer(buffer) do |zip|
        segments = zip.filter_map do |e|
          n = e.name.to_s
          next if n.blank?
          n = n.sub(%r{\A/+}, "")
          next if n.blank?
          n.split("/", 2).first.presence
        end
        seg = segments.uniq.size == 1 ? segments.first : nil
        return "" if seg.blank?
        return "" if %w[service checker].include?(seg)
        "#{seg}/"
      end
    rescue Zip::Error
      ""
    end

    def readme_candidates
      [ "README.md", "readme.md", "README", "readme" ]
    end

    def license_candidates
      [ "LICENSE", "LICENSE.txt", "LICENSE.md", "LICENCE", "LICENCE.txt", "COPYING", "COPYING.txt" ]
    end

    def read_first(zip, root_prefix, candidates)
      candidates.each do |name|
        entry = zip.find_entry(File.join(root_prefix, name))
        next unless entry
        return [ name, read_small_entry(entry, max_bytes: 512 * 1024) ]
      end
      [ nil, nil ]
    end

    def copy_tree(zip, zos, from_prefix, to_prefix, exclude_rel_prefixes: [])
      found = false
      zip.each do |entry|
        next unless entry.name.start_with?(from_prefix)
        rel = safe_rel_path(entry.name.sub(from_prefix, ""))
        next if rel.blank?
        next if exclude_rel_prefixes.any? { |p| rel == p.to_s.delete_suffix("/") || rel.start_with?(p.to_s) }
        next if rel.start_with?(".git/")
        yield(rel) if block_given?
        if entry.directory?
          dir_rel = rel.end_with?("/") ? rel : "#{rel}/"
          zos.put_next_entry(to_prefix + dir_rel)
        else
          guard_zip_entry!(entry)
          zos.put_next_entry(to_prefix + rel)
          zos.write(read_entry_limited(entry))
          found = true
        end
      end
      found
    end

    def read_small_entry(entry, max_bytes:)
      data = +""
      entry.get_input_stream do |io|
        while (chunk = io.read(16 * 1024))
          break if chunk.empty?
          data << chunk
          raise Error, "служебный файл в архиве слишком большой" if data.bytesize > max_bytes
        end
      end
      data
    end

    def safe_rel_path(rel)
      s = rel.to_s.tr("\\", "/")
      return nil if s.empty?
      return nil if s.include?("\0")
      s = s.sub(%r{\A/+}, "")
      s = s.sub(%r{/+\z}, "")
      return nil if s.empty?

      segments = s.split("/")
      return nil if segments.any?(&:blank?)
      return nil if segments.any? { |seg| seg == "." || seg == ".." }

      segments.join("/")
    end

    def guard_zip_entry!(entry)
      @files += 1
      raise Error, "слишком много файлов в архиве" if @files > MAX_FILES

      size = entry.size.to_i
      raise Error, "файл в архиве слишком большой (#{size} байт)" if size > MAX_ENTRY_BYTES
      @total_bytes += size
      raise Error, "архив слишком большой (суммарно #{ @total_bytes } байт)" if @total_bytes > MAX_TOTAL_BYTES
    end

    def read_entry_limited(entry)
      # Если zip сообщает size — guard_zip_entry! уже отработал.
      data = +""
      entry.get_input_stream do |io|
        while (chunk = io.read(16 * 1024))
          break if chunk.empty?
          data << chunk
          raise Error, "файл в архиве слишком большой" if data.bytesize > MAX_ENTRY_BYTES
        end
      end
      data
    end
  end
end
