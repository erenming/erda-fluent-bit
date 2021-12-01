function add_cpuset(tag, timestamp, record)
    new_record = record

    -- container's cpuset file path to extract containerID
    local cname = new_record["__id"]

    -- get cpuset from shared emptyDir
    local root_path = "/erda/containers/"
    file = io.open(root_path .. cname .. "/cpuset", "r")
    new_record["cpuset"] = file:read()
    file:close()

    return 1, timestamp, new_record
end
