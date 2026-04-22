require "../spec_helper"

describe Open101::WizServer::Realm do
  it "assigns players to first available zone" do
    realm = Open101::WizServer::Realm.new(100_u32, 2_u32, 1_u32)
    one = Open101::WizServer::Character.new(1_u32, 10_u32, "A")
    two = Open101::WizServer::Character.new(2_u32, 11_u32, "B")
    realm.assign(one).not_nil!.id.should eq(0_u32)
    realm.assign(two).not_nil!.id.should eq(1_u32)
  end
end
