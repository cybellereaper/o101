require "../spec_helper"

describe O101::WizServer::Realm do
  it "assigns players into least-populated zones" do
    realm = O101::WizServer::Realm.new(2, 2, 3)

    z1 = realm.assign_zone("a")
    z2 = realm.assign_zone("b")

    [z1.id, z2.id].uniq.size.should eq(2)
  end

  it "enforces max players" do
    realm = O101::WizServer::Realm.new(1, 10, 1)
    realm.assign_zone("a")
    expect_raises(O101::Error) { realm.assign_zone("b") }
  end
end
