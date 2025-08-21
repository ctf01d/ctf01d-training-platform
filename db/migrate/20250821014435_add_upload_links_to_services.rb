class AddUploadLinksToServices < ActiveRecord::Migration[8.0]
  def change
    add_column :services, :service_archive_url, :string
    add_column :services, :checker_archive_url, :string
    add_column :services, :writeup_url, :string
  end
end

