class AddLocalArchivesToServices < ActiveRecord::Migration[8.0]
  def change
    change_table :services, bulk: true do |t|
      # Локальные копии архивов сервиса и чекера + метаданные
      t.string  :service_local_path
      t.integer :service_local_size
      t.string  :service_local_sha256
      t.datetime :service_downloaded_at

      t.string  :checker_local_path
      t.integer :checker_local_size
      t.string  :checker_local_sha256
      t.datetime :checker_downloaded_at
    end
  end
end
