require "../spec_helper"

describe O101::MessageSorter::Sorter do
  it "extracts, deduplicates, and sorts messages" do
    input = <<-XML
      <root service="LoginService" id="42">
        <message>Ping</message>
        <message>Pong</message>
        <message>Ping</message>
      </root>
    XML

    result = O101::MessageSorter::Sorter.new.sort(input)
    result.service_id.should eq("42")
    result.messages.should eq(["Ping", "Pong"])
  end

  it "fails when service metadata is absent" do
    expect_raises(O101::ParseError) { O101::MessageSorter::Sorter.new.sort("<message>Ping</message>") }
  end
end
