require "../spec_helper"

# Patcher network behavior is covered by integration tests in deployment environments.
describe Open101::Patcher::Runner do
  it "is constructible" do
    dir = File.join(Dir.tempdir, "open101-spec-#{Random.rand(1_000_000)}")
    Dir.mkdir_p(dir)
    store = Open101::State::Store.new(File.join(dir, "state.json"))
    config = Open101::Patcher::Config.new("https://example.com/patch-info.json", dir, store)
    Open101::Patcher::Runner.new(config).should be_a(Open101::Patcher::Runner)
    FileUtils.rm_rf(dir)
  end
end
