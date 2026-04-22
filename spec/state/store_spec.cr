require "../spec_helper"

describe O101::State::Store do
  it "returns empty snapshot when file does not exist" do
    store = O101::State::Store.new("/tmp/o101-state-missing-#{Random.rand}.json")
    store.load.applied_files.should be_empty
  end

  it "persists and reloads snapshots" do
    path = "/tmp/o101-state-#{Random.rand}.json"
    store = O101::State::Store.new(path)
    snapshot = O101::State::Snapshot.new(version: "2.0.0", applied_files: ["Bin/app.bin"], updated_at: Time.utc.to_rfc3339)

    store.save(snapshot)
    loaded = store.load

    loaded.version.should eq("2.0.0")
    loaded.applied_files.should eq(["Bin/app.bin"])
  ensure
    File.delete(path.not_nil!) if File.exists?(path.not_nil!)
  end
end
