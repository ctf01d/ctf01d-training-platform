# frozen_string_literal: true

class AddCtf01dTrainingToServices < ActiveRecord::Migration[8.0]
  def change
    add_column :services, :ctf01d_training, :jsonb, default: {}, null: false
  end
end

