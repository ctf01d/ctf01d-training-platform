# frozen_string_literal: true

require "yaml"
require "fileutils"
require "securerandom"
require "stringio"
require "net/http"
require "uri"
require "base64"

module Ctf01d
  class ExportError < StandardError; end

  # Сервис: собрать архив с конфигом и ресурсами для ctf01d
  # Важно: сервис не берёт данные из БД напрямую — всё передаётся параметрами,
  # чтобы явно управлять матчингом id/ip/логотипов/скриптов.
  #
  # Использование (пример):
  #   result = Ctf01d::ExportZip.call(
  #     game: { id: 'mygame01', name: 'My Game', start_utc: Time.utc(2025,10,1,9,0,0), end_utc: Time.utc(2025,10,1,19,0,0),
  #             coffee_break_start_utc: nil, coffee_break_end_utc: nil,
  #             flag_ttl_min: 1, basic_attack_cost: 1, defence_cost: 1.0 },
  #     scoreboard: { port: 8080, htmlfolder: './html', random: false },
  #     teams: [ { id: 't01', name: 'Team #1', active: true, ip_address: '10.0.1.1', logo_rel: './html/images/teams/team01.png', logo_src: '/abs/path/team01.png' } ],
  #     checkers: [ { id: 'service1', name: 'Service1', enabled: true, script_wait: 5, round_sleep: 15, script_rel: './checker.py', files: [ { src: '/abs/path/checker.py', rel: 'checker.py' } ] } ],
  #     options: { prefix: 'ctf01d_package', include_html: true, html_source_path: Rails.root.join('ctf01d','data_sample','html').to_s,
  #               include_compose: true, compose_project: 'my-first-game' }
  #   )
  #   File.binwrite('/tmp/mygame.zip', result[:data])
  class ExportZip
    # Параметры:
    # - game: { id:, name:, start_utc:, end_utc:, coffee_break_start_utc:, coffee_break_end_utc:, flag_ttl_min:, basic_attack_cost:, defence_cost: }
    # - scoreboard: { port:, htmlfolder:, random: }
    # - teams: [ { id:, name:, active:, ip_address:, logo_rel:, logo_src: } ]
    # - checkers: [ { id:, name:, enabled:, script_wait:, round_sleep:, script_rel:, files: [ { src:, rel: } ] } ]
    # - options: { prefix:, include_html: true/false, html_source_path:, include_compose: true/false, compose_project: }
    def self.call(game:, scoreboard:, teams:, checkers:, options: {})
      new(game, scoreboard, teams, checkers, options).call
    end

    def initialize(game, scoreboard, teams, checkers, options)
      @game = game
      @scoreboard = scoreboard
      @teams = teams || []
      @checkers = checkers || []
      @options = {
        prefix: "ctf01d_package_#{SecureRandom.hex(4)}",
        include_html: true,
        html_source_path: File.join(Rails.root.to_s, "ctf01d", "data_sample", "html"),
        include_compose: false,
        compose_project: "ctf01d_game"
      }.merge(options || {})
      @errors = []
    end

    def call
      hydrate_checkers_from_bundles!
      validate_inputs!

      Dir.mktmpdir("ctf01d_export_") do |tmpdir|
        root = File.join(tmpdir, @options[:prefix])
        data_dir = File.join(root, "data")
        html_target = File.join(data_dir, "html")
        FileUtils.mkdir_p(data_dir)

        html_source = @options[:html_source_path]
        if @options[:include_html]
          unless html_source && Dir.exist?(html_source)
            html_source = build_fallback_html(tmpdir)
          end
          copy_tree!(html_source, html_target, required: false)
        end

        # 2) Логотипы команд в соответствии с logo_rel
        downloads_dir = File.join(tmpdir, "downloads")
        FileUtils.mkdir_p(downloads_dir)
        ensure_team_logos!(data_dir, downloads_dir)

        # 3) Чекеры (файлы + папки)
        materialize_checkers!(data_dir)

        # 3.1) Архивы сервисов (bundle: service/ + checker/)
        materialize_service_archives!(root)

        # 4) config.yml
        cfg_path = File.join(data_dir, "config.yml")
        File.write(cfg_path, build_yaml_config)

        materialize_warnings!(root)

        # 5) docker-compose.yml (если включено)
        if @options[:include_compose]
          File.write(File.join(root, "docker-compose.yml"), compose_yml)
        end

        # 6) Упаковка в zip
        zip_data = pack_zip(root)
        return { filename: "#{@options[:prefix]}.zip", data: zip_data, size: zip_data.bytesize }
      end
    end

    private
    def require_zip!
      require "zip"
    rescue LoadError
      raise ExportError, "Гем rubyzip не установлен. Добавьте gem 'rubyzip' в Gemfile и выполните bundle install"
    end

    def hydrate_checkers_from_bundles!
      @checkers.each do |c|
        bundle_path = c[:bundle_path].to_s
        next if bundle_path.blank?
        unless File.file?(bundle_path)
          raise ExportError, "bundle_path не найден: #{bundle_path}"
        end
        c[:script_wait] = 10 if c[:script_wait].to_i <= 0
        c[:round_sleep] = [ c[:round_sleep].to_i, c[:script_wait].to_i * 3 ].max
        if c[:checker_from_bundle]
          c[:script_rel] = detect_checker_entrypoint(bundle_path).presence || "./checker.py" if c[:script_rel].to_s.strip.empty?
        else
          c[:script_rel] = "./checker.py" if c[:script_rel].to_s.strip.empty?
        end
      end
    end

    def validate_inputs!
      # game
      gid = @game[:id].to_s
      raise ExportError, "game.id обязателен" if gid.empty?
      raise ExportError, "game.id должен соответствовать [a-z0-9]+" unless gid =~ /\A[a-z0-9]+\z/

      %i[name start_utc end_utc flag_ttl_min basic_attack_cost defence_cost].each do |k|
        raise ExportError, "game.#{k} обязателен" if @game[k].nil? || (@game[k].respond_to?(:empty?) && @game[k].empty?)
      end
      raise ExportError, "game.end_utc должен быть позже game.start_utc" unless @game[:end_utc] > @game[:start_utc]
      ttl = @game[:flag_ttl_min].to_i
      raise ExportError, "game.flag_ttl_min должен быть в диапазоне 1..25" unless ttl.between?(1, 25)
      bac = @game[:basic_attack_cost].to_i
      raise ExportError, "game.basic_attack_cost должен быть в диапазоне 1..500" unless bac.between?(1, 500)

      # scoreboard
      port = @scoreboard[:port].to_i
      raise ExportError, "scoreboard.port должен быть в диапазоне 11..65535" unless port.between?(11, 65_535)
      htmlfolder = @scoreboard[:htmlfolder].to_s
      raise ExportError, "scoreboard.htmlfolder должен быть './html' или относительным путём" if htmlfolder.empty?

      # teams
      raise ExportError, "teams: требуется хотя бы одна команда" if @teams.empty?
      team_ids = {}
      team_ips = {}
      @teams.each do |t|
        raise ExportError, "team.id обязателен" if t[:id].to_s.empty?
        raise ExportError, "дубликат team.id: #{t[:id]}" if team_ids[t[:id]]
        team_ids[t[:id]] = true
        ip = t[:ip_address].to_s
        raise ExportError, "team #{t[:id]}: ip_address обязателен" if ip.empty?
        raise ExportError, "team #{t[:id]}: ip_address ожидается IPv4" unless ip =~ /\A(?:\d{1,3}\.){3}\d{1,3}\z/
        raise ExportError, "дубликат ip_address: #{ip}" if team_ips[ip]
        team_ips[ip] = true
        # logo_rel / logo_src / logo_url валидируются позже при материализации
      end

      # checkers (хотя бы один)
      raise ExportError, "checkers: требуется хотя бы один сервис" if @checkers.empty?
      chk_ids = {}
      @checkers.each do |c|
        cid = normalize_id(c[:id])
        raise ExportError, "checker.id обязателен" if cid.empty?
        raise ExportError, "дубликат checker.id: #{cid}" if chk_ids[cid]
        chk_ids[cid] = true
        w = c[:script_wait].to_i
        s = c[:round_sleep].to_i
        raise ExportError, "checker #{cid}: script_wait >= 5" if w < 5
        raise ExportError, "checker #{cid}: round_sleep должен быть >= script_wait * 3" if s < (w * 3)
        raise ExportError, "checker #{cid}: script_rel обязателен" if c[:script_rel].to_s.empty?
      end
    end

    def build_yaml_config
      game = {
        "id" => @game[:id].to_s,
        "name" => @game[:name].to_s,
        "start" => @game[:start_utc].utc.strftime("%Y-%m-%d %H:%M:%S"),
        "end" => @game[:end_utc].utc.strftime("%Y-%m-%d %H:%M:%S"),
        "flag_timelive_in_min" => @game[:flag_ttl_min].to_i,
        "basic_costs_stolen_flag_in_points" => @game[:basic_attack_cost].to_i,
        "cost_defence_flag_in_points" => @game[:defence_cost].to_f
      }
      if @game[:coffee_break_start_utc] && @game[:coffee_break_end_utc]
        game["coffee_break_start"] = @game[:coffee_break_start_utc].utc.strftime("%Y-%m-%d %H:%M:%S")
        game["coffee_break_end"] = @game[:coffee_break_end_utc].utc.strftime("%Y-%m-%d %H:%M:%S")
      end

      scoreboard = {
        "port" => @scoreboard[:port].to_i,
        "htmlfolder" => @scoreboard[:htmlfolder].to_s,
        "random" => !!@scoreboard[:random]
      }

      checkers = @checkers.map do |c|
        {
          "id" => normalize_id(c[:id]),
          "service_name" => c[:name].to_s,
          "enabled" => !!c[:enabled],
          "script_path" => c[:script_rel].to_s,
          "script_wait_in_sec" => c[:script_wait].to_i,
          "time_sleep_between_run_scripts_in_sec" => c[:round_sleep].to_i
        }
      end

      teams = @teams.map do |t|
        row = {
          "id" => t[:id].to_s,
          "name" => t[:name].to_s,
          "active" => !!t[:active],
          "logo" => t[:logo_rel].to_s,
          "ip_address" => t[:ip_address].to_s
        }

        extras = (t[:ctf01d_extra] || {}).to_h
        extras.each do |k, v|
          key = k.to_s.sub(/\Actf01d_/, "")
          row[key] = v
        end

        row
      end

      data = {
        "game" => game,
        "scoreboard" => scoreboard,
        "checkers" => checkers,
        "teams" => teams
      }

      header = [
        "## Combined config for ctf01d",
        "# Автогенерация CRM: не редактировать вручную, лучше пересобрать архив."
      ].join("\n")

      [ header, "", YAML.dump(data) ].join("\n")
    end

    def detect_checker_entrypoint(bundle_path)
      require_zip!
      rel_files = []
      Zip::File.open(bundle_path) do |zip|
        zip.each do |e|
          next if e.name.to_s.end_with?("/")
          name = e.name.to_s
          next unless name =~ %r{(^|/)checker/}
          rel = name.sub(%r{\A.*?checker/}, "")
          next if rel.blank?
          rel_files << rel
        end
      end
      return nil if rel_files.empty?

      candidates = %w[
        checker.py checker.rb checker.pl checker.sh checker.php checker.go checker.cr checker.js checker.ts
      ]

      pick_by_basename = ->(basename) do
        matches = rel_files.select { |rel| File.basename(rel) == basename }
        matches.min_by { |rel| [ rel.count("/"), rel.length ] }
      end

      chosen = nil
      candidates.each do |basename|
        chosen = pick_by_basename.call(basename)
        break if chosen
      end

      unless chosen
        # Частый кейс: любой файл на верхнем уровне checker/
        top_level = rel_files.select { |rel| !rel.include?("/") }
        preferred = top_level.select { |rel| File.basename(rel).start_with?("checker.") || File.basename(rel) == "checker" }
        chosen = (preferred.presence || top_level.presence || rel_files).first
      end

      "./#{chosen}"
    rescue Zip::Error => e
      raise ExportError, "не удалось прочитать zip #{File.basename(bundle_path)}: #{e.message}"
    end

    def ensure_team_logos!(data_dir, downloads_dir)
      @teams.each do |t|
        # Подготовим относительный путь, если не задан
        if t[:logo_rel].to_s.strip.empty?
          t[:logo_rel] = "./html/images/teams/#{safe_team_id(t[:id])}.svg"
        end

        # Определим источник: локальный файл, URL, data:image, либо сгенерируем SVG аватар
        src = t[:logo_src]
        # 0) Если logo_url указывает на локальный путь в приложении (/uploads/..., /img/...)
        unless src && File.file?(src)
          local = t[:logo_url].to_s
          if local.present? && !local.start_with?("http://", "https://", "data:image")
            candidate = safe_public_asset_path(local)
            src = candidate if candidate
          end
        end
        unless src && File.file?(src)
          if t[:logo_url].to_s.start_with?("http://", "https://")
            src = download_url_to_file(t[:logo_url].to_s, downloads_dir, prefer_name: safe_team_id(t[:id]))
          elsif t[:logo_url].to_s.start_with?("data:image")
            src = write_data_url_to_file(t[:logo_url].to_s, downloads_dir, prefer_name: safe_team_id(t[:id]))
          else
            # Генерация SVG по имени команды
            src = generate_svg_logo_to_file((t[:name].presence || t[:id].to_s), downloads_dir, prefer_name: safe_team_id(t[:id]))
            # если в logo_rel указан .png — заменим на .svg, чтобы совпадало с содержимым
            if File.extname(t[:logo_rel].to_s).downcase == ".png"
              t[:logo_rel] = t[:logo_rel].to_s.sub(/\.png\z/i, ".svg")
            end
          end
        end

        # Если расширение у logo_rel не совпадает с реальным файлом — приведём к расширению файла.
        src_ext = File.extname(src.to_s).downcase
        if src_ext.present? && File.extname(t[:logo_rel].to_s).downcase != src_ext
          t[:logo_rel] = t[:logo_rel].to_s.sub(/\.[a-z0-9]+\z/i, src_ext)
        end

        target = File.join(data_dir, t[:logo_rel].to_s)
        FileUtils.mkdir_p(File.dirname(target))
        FileUtils.cp(src, target)
      end
    end

    def safe_team_id(val)
      val.to_s.downcase.gsub(/[^a-z0-9]+/, "_").gsub(/\A_+|_+\z/, "")
    end

    def safe_public_asset_path(urlish)
      rel = urlish.to_s.tr("\\", "/").sub(%r{\A/+}, "")
      return nil if rel.blank?
      return nil if rel.include?("\0")
      return nil unless rel.start_with?("uploads/", "img/")

      public_root = Rails.root.join("public").cleanpath
      path = Rails.root.join("public", rel).cleanpath
      return nil unless path.to_s.start_with?(public_root.to_s + File::SEPARATOR)
      return nil unless File.file?(path)
      path.to_s
    end

    def write_data_url_to_file(data_url, dir, prefer_name: "logo")
      # Поддержка base64 и utf8 (url-encoded) вариантов
      if (m = data_url.match(/\Adata:(image\/[a-zA-Z0-9.+-]+);base64,(.+)\z/))
        mime = m[1]
        payload = m[2]
        bytes = Base64.decode64(payload)
        ext = ext_from_mime(mime)
        path = File.join(dir, "#{prefer_name}#{ext}")
        File.binwrite(path, bytes)
        path
      elsif (m = data_url.match(/\Adata:(image\/[a-zA-Z0-9.+-]+);utf8,(.+)\z/))
        mime = m[1]
        encoded = m[2]
        raw = URI.decode_www_form_component(encoded)
        ext = ext_from_mime(mime)
        path = File.join(dir, "#{prefer_name}#{ext}")
        File.write(path, raw)
        path
      else
        raise ExportError, "некорректный data:image (ожидался base64 или utf8)"
      end
    end

    def download_url_to_file(url, dir, prefer_name: "logo")
      uri = URI.parse(url)
      redirects = 0
      res = nil
      max_bytes = 5 * 1024 * 1024
      loop do
        raise ExportError, "слишком много редиректов при скачивании logo" if redirects > 5
        http = Net::HTTP.new(uri.host, uri.port)
        http.use_ssl = (uri.scheme == "https")
        http.open_timeout = 5
        http.read_timeout = 10
        configure_ssl!(http) if http.use_ssl?
        req = Net::HTTP::Get.new(uri.request_uri)
        redirect_location = nil
        tmp = File.join(dir, "#{prefer_name}.part")
        FileUtils.rm_f(tmp)
        http.request(req) do |r|
          res = r
          if r.is_a?(Net::HTTPRedirection)
            redirect_location = r["location"].to_s
            next
          end
          next unless r.is_a?(Net::HTTPSuccess)
          bytes = 0
          File.open(tmp, "wb") do |f|
            r.read_body do |chunk|
              next if chunk.nil? || chunk.empty?
              bytes += chunk.bytesize
              raise ExportError, "logo слишком большой" if bytes > max_bytes
              f.write(chunk)
            end
          end
        end
        if redirect_location.present?
          FileUtils.rm_f(tmp)
          raise ExportError, "редирект без Location при скачивании logo" if redirect_location.blank?
          uri = URI.join(uri.to_s, redirect_location)
          redirects += 1
          next
        end
        break if res
      end
      raise ExportError, "не удалось скачать logo: HTTP #{res.code}" unless res.is_a?(Net::HTTPSuccess)
      mime = res["content-type"].to_s.split(";").first
      ext = ext_from_mime(mime)
      path = File.join(dir, "#{prefer_name}#{ext}")
      tmp = File.join(dir, "#{prefer_name}.part")
      FileUtils.mv(tmp, path) if File.file?(tmp)
      path
    rescue OpenSSL::SSL::SSLError => e
      raise ExportError, "SSL ошибка при скачивании logo: #{e.message}"
    end

    def configure_ssl!(http)
      store = OpenSSL::X509::Store.new
      store.set_default_paths
      store.flags = 0 if store.respond_to?(:flags=)
      http.verify_mode = OpenSSL::SSL::VERIFY_PEER
      http.cert_store = store
    end

    def ext_from_mime(mime)
      case mime
      when "image/png" then ".png"
      when "image/jpeg", "image/jpg" then ".jpg"
      when "image/svg+xml" then ".svg"
      when "image/gif" then ".gif"
      else ".png"
      end
    end

    def generate_svg_logo_to_file(text, dir, prefer_name: "logo")
      # используем хелпер для генерации SVG-аватара
      data_url = ApplicationController.helpers.svg_data_avatar(text, size: 128)
      write_data_url_to_file(data_url, dir, prefer_name: prefer_name)
    end

    def materialize_checkers!(data_dir)
      @checkers.each do |c|
        cid = normalize_id(c[:id])
        dir = File.join(data_dir, "checker_#{cid}")
        FileUtils.mkdir_p(dir)

        bundle_path = c[:bundle_path].to_s
        if bundle_path.present? && c[:checker_from_bundle]
          extracted = extract_checker_dir_from_bundle!(bundle_path: bundle_path, dest_dir: dir)
          write_dummy_checker!(dest_dir: dir, cid: cid) unless extracted
          next
        end
        if bundle_path.present? && !c[:checker_from_bundle]
          write_dummy_checker!(dest_dir: dir, cid: cid)
          next
        end

        files = (c[:files] || [])
        files = [ { src: nil, rel: "checker.py" } ] if files.empty?
        files.each do |f|
          src = f[:src].to_s
          rel = f[:rel].presence || (File.file?(src) ? File.basename(src) : "checker.py")
          dest = safe_join(dir, rel)
          FileUtils.mkdir_p(File.dirname(dest))
          if File.file?(src)
            FileUtils.cp(src, dest)
          else
            File.write(dest, "#!/usr/bin/env python3\nprint('dummy checker for #{cid}')\n")
          end
        end
      end
    end

    def materialize_service_archives!(root_dir)
      dir = File.join(root_dir, "archives", "services")
      FileUtils.mkdir_p(dir)
      @checkers.each do |c|
        bundle_path = c[:bundle_path].to_s
        next if bundle_path.blank?
        next unless File.file?(bundle_path)
        cid = normalize_id(c[:id])
        FileUtils.cp(bundle_path, File.join(dir, "#{cid}.zip"))
      end
    end

    def extract_checker_dir_from_bundle!(bundle_path:, dest_dir:)
      require_zip!
      extracted_any = false
      Zip::File.open(bundle_path) do |zip|
        zip.each do |e|
          name = e.name.to_s
          next unless name =~ %r{(^|/)checker/}
          extracted_any = true
          rel = name.sub(%r{\A.*?checker/}, "")
          next if rel.blank?
          target = safe_join(dest_dir, rel)
          if name.end_with?("/")
            FileUtils.mkdir_p(target)
            next
          end
          FileUtils.mkdir_p(File.dirname(target))
          e.extract(target) { true }
        end
      end
      extracted_any
    rescue Zip::Error => e
      raise ExportError, "не удалось прочитать zip #{File.basename(bundle_path)}: #{e.message}"
    end

    def write_dummy_checker!(dest_dir:, cid:)
      path = File.join(dest_dir, "checker.py")
      return if File.file?(path)
      File.write(path, "#!/usr/bin/env python3\nprint('dummy checker for #{cid}')\n")
    end

    def safe_join(base, rel)
      clean = rel.to_s.tr("\\", "/").sub(%r{\A/+}, "")
      raise ExportError, "некорректный путь в архиве: #{rel}" if clean.blank?
      raise ExportError, "некорректный путь в архиве: #{rel}" if clean.include?("\0")

      segments = clean.split("/")
      raise ExportError, "некорректный путь в архиве: #{rel}" if segments.any?(&:blank?)
      raise ExportError, "некорректный путь в архиве: #{rel}" if segments.any? { |s| s == "." || s == ".." }

      File.join(base, segments.join("/"))
    end

    def materialize_warnings!(root_dir)
      warnings = Array(@options[:warnings]).map(&:to_s).map(&:strip).reject(&:blank?)
      return if warnings.empty?
      File.write(File.join(root_dir, "EXPORT_WARNINGS.txt"), warnings.join("\n"))
    end

    def copy_tree!(src, dst, required: false)
      unless Dir.exist?(src)
        raise ExportError, "исходная папка не найдена: #{src}" if required
        return
      end
      FileUtils.mkdir_p(dst)
      Dir.chdir(src) do
        Dir.glob("**/*", File::FNM_DOTMATCH).each do |entry|
          next if [ ".", ".." ].include?(entry)
          s = File.join(src, entry)
          d = File.join(dst, entry)
          if File.directory?(s)
            FileUtils.mkdir_p(d)
          else
            FileUtils.mkdir_p(File.dirname(d))
            FileUtils.cp(s, d)
          end
        end
      end
    end

    def build_fallback_html(tmpdir)
      dir = File.join(tmpdir, "fallback_html")
      FileUtils.mkdir_p(dir)
      File.write(File.join(dir, "index-template.html"), <<~HTML)
        <!doctype html>
        <html>
          <head>
            <meta charset="utf-8">
            <title>ctf01d scoreboard</title>
          </head>
          <body>
            <h1>ctf01d scoreboard placeholder</h1>
            <p>HTML не найден в репозитории, сгенерирован шаблон по умолчанию.</p>
          </body>
        </html>
      HTML
      teams_dir = File.join(dir, "images", "teams")
      FileUtils.mkdir_p(teams_dir)
      png = Base64.decode64("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+ZzoAAAAASUVORK5CYII=")
      10.times do |i|
        n = i + 1
        File.binwrite(File.join(teams_dir, format("team%02d.png", n)), png)
      end
      dir
    end

    def compose_yml
      project = @options[:compose_project].to_s
      <<~YML
      version: '3'

      services:
        ctf01d_jury:
          container_name: ctf01d_jury_#{project}
          image: sea5kg/ctf01d:latest
          volumes:
            - "./data:/usr/share/ctf01d"
          environment:
            CTF01D_WORKDIR: "/usr/share/ctf01d"
          ports:
            - "#{@scoreboard[:port]}:#{@scoreboard[:port]}"
          networks:
            - ctf01d_net

      networks:
        ctf01d_net:
          driver: bridge
      YML
    end

    def pack_zip(root_dir)
      require_zip!
      buffer = Zip::OutputStream.write_buffer do |zos|
        Dir.chdir(File.dirname(root_dir)) do
          base = File.basename(root_dir)
          Dir.glob("#{base}/**/*", File::FNM_DOTMATCH).each do |path|
            next if path.end_with?("/.") || path.end_with?("/..")
            if File.directory?(path)
              zos.put_next_entry("#{path}/")
            else
              zos.put_next_entry(path)
              zos.write(File.binread(path))
            end
          end
        end
      end
      buffer.rewind
      buffer.read
    end

    def normalize_id(val)
      val.to_s.downcase.gsub(/[^a-z0-9]+/, "_").gsub(/\A_+|_+\z/, "")
    end
  end
end
