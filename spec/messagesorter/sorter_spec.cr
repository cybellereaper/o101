require "../spec_helper"

describe Open101::MessageSorter::Sorter do
  it "extracts and sorts deduplicated tags" do
    input = <<-TXT
    <ServiceID TYPE="UBYT">42</ServiceID>
    <ProtocolType TYPE="STR">LoginService</ProtocolType>
    </_ProtocolInfo>
    <RECORD>ignored</RECORD>
    <Foo>
    <Bar>
    <Foo>
    TXT

    result = Open101::MessageSorter::Sorter.process(input)
    result.service_id.should eq("42")
    result.messages.should eq(["Bar", "Foo"])
  end
end
