class CreateUniversities < ActiveRecord::Migration[8.0]
  def change
    create_table :universities do |t|
      t.string :name
      t.string :site_url

      t.timestamps
    end
  end
end
