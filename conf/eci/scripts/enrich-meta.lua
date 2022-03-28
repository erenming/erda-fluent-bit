function add_cpuset(tag, timestamp, record)
    new_record = record

    -- container's cpuset file path to extract containerID
    local cname = new_record["__pri_id"]
    if cname == nil then
        print(dump(new_record))
        return -1, timestamp, new_record
    end

    -- get cpuset from shared emptyDir
    local root_path = "/erda/containers/"
    --local root_path = "testdata/eci/containers/"
    local file = io.open(root_path .. cname .. "/cpuset", "r")
    if file == nil then
        print(dump(new_record))
        return -1, timestamp, new_record
    end
    new_record["__pri_cpuset"] = file:read()
    file:close()

    return 1, timestamp, new_record
end

function dump(o)
    if type(o) == 'table' then
        local s = '{ '
        for k, v in pairs(o) do
            if type(k) ~= 'number' then
                k = '"' .. k .. '"'
            end
            s = s .. '[' .. k .. '] = ' .. dump(v) .. ','
        end
        return s .. '} '
    else
        return tostring(o)
    end
end