function add_cpuset(tag, timestamp, record)
    new_record = record

    -- container's cpuset file path to extract containerID
    local cname = new_record["__pri_id"]

    -- get cpuset from shared emptyDir
    local root_path = "/erda/containers/"
    --local root_path = "testdata/eci/containers/"
    file = io.open(root_path .. cname .. "/cpuset", "r")
    new_record["__pri_cpuset"] = file:read()
    file:close()

    return 1, timestamp, new_record
end
