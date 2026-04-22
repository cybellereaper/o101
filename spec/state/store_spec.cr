require "../spec_helper"

describe Open101::State::Store do
  it "round-trips snapshots" do
    dir = File.join(Dir.tempdir, "open101-spec-#{Random.rand(1_000_000)}")
    Dir.mkdir_p(dir)
    path = File.join(dir, "state.json")
    store = Open101::State::Store.new(path)
    snapshot = Open101::State::Snapshot.new("1.0", {"a" => Open101::State::FileInfo.new(3_i64, "abc")})
    store.save(snapshot)
    loaded = store.load
    loaded.version.should eq("1.0")
    loaded.files["a"].size.should eq(3)
    FileUtils.rm_rf(dir)
  end
end
