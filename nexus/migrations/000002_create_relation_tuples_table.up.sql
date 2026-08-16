CREATE TABLE relation_tuples (
    namespace VARCHAR(128) NOT NULL,
    object VARCHAR(128) NOT NULL,
    relation VARCHAR(64) NOT NULL,
    
    subject_namespace VARCHAR(128) NOT NULL,
    subject_object VARCHAR(128) NOT NULL,
    subject_relation VARCHAR(64) NOT NULL,
    
    -- ABAC Integration
    caveat_name VARCHAR(128) REFERENCES caveats(name),
    caveat_context JSONB, 
    
    commit_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (namespace, object, relation, subject_namespace, subject_object, subject_relation)
);

-- Index for answering: "Who has access to object X?" or "Does Subject Y have access to Object X?"
CREATE INDEX idx_tuples_object_relation 
ON relation_tuples (namespace, object, relation);

-- Index for answering: "What objects does Subject Y have access to?"
CREATE INDEX idx_tuples_subject 
ON relation_tuples (subject_namespace, subject_object, subject_relation);

-- Index to quickly find tuples that depend on a specific caveat
CREATE INDEX idx_tuples_caveats
ON relation_tuples (caveat_name) WHERE caveat_name IS NOT NULL;
