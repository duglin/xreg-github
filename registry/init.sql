-- DROP DATABASE IF EXISTS registry ;
-- CREATE DATABASE registry ;
-- USE registry ;
-- ^^ OLD STUF

-- MySQL config requirements:
-- sql_mode:
--   ANSI_QUOTES        <- enabled
--   ONLY_FULL_GROUP_BY <- disabled

/*
Notes:
SID -> System generated ID, usually primary key. Assumed to be globally
unique so that when searching we don't need to combine it with any other
scoping column to provide uniqueness
UID -> User provided ID. Only needs to be unique within its parent scope.
This is why we use SID for cross-table joins/links.
The code doesn't do delete propagation. Instead, the code will delete
whichever resource was asked to be deleted and then the DB triggers
will delete all necessarily (related) rows/resources as needed. So,
deleting a row from the "Registry" table should delete ALL other resources
in all other tables automatically.
The "Props" table holds all properties for all entities rather than
having property specific columns in the appropriate tables. No idea which
is easier/faster but having it all in one table made things a lot easier
for filtering/searching. But we can switch it if needed at some point. This
also means that all properties (including extensions) are processed the
same way... via the generic Get/Set methods.
*/


SET GLOBAL sql_mode = 'ANSI_QUOTES' ;
SET sql_mode = 'ANSI_QUOTES' ;

CREATE TABLE Registries (
    SID     VARCHAR(255) NOT NULL,  # System ID
    UID     VARCHAR(255) NOT NULL,  # User defined
    Attributes  JSON,               # Until we use the Attributes table

    PRIMARY KEY (SID),
    UNIQUE INDEX (UID)
);

CREATE TRIGGER RegistryTrigger BEFORE DELETE ON Registries
FOR EACH ROW
BEGIN
    DELETE FROM Props    WHERE EntitySID=OLD.SID @
    DELETE FROM "Groups" WHERE RegistrySID=OLD.SID @
    DELETE FROM Models   WHERE RegistrySID=OLD.SID @
END ;

CREATE TABLE Models (
    RegistrySID VARCHAR(64) NOT NULL,

    PRIMARY KEY (RegistrySID)
);

CREATE TRIGGER ModelsTrigger BEFORE DELETE ON Models
FOR EACH ROW
BEGIN
    DELETE FROM ModelEntities WHERE RegistrySID=OLD.RegistrySID @
    DELETE FROM "Schemas"     WHERE RegistrySID=OLD.RegistrySID @
END ;

CREATE TABLE "Schemas" (
    RegistrySID  VARCHAR(64) NOT NULL,
    "Schema"     VARCHAR(255) NOT NULL,

    PRIMARY KEY(RegistrySID, "Schema"),
    INDEX (RegistrySID)
);

CREATE TABLE ModelEntities (        # Group or Resource (no parent=Group)
    SID               VARCHAR(64),        # my System ID
    RegistrySID       VARCHAR(64),
    ParentSID         VARCHAR(64),        # ID of parent ModelEntity

    Singular          VARCHAR(64),
    Plural            VARCHAR(64),
    Attributes        JSON,               # Until we use the Attributes table

    MaxVersions       INT,      # For Resources
    SetVersionId      BOOL,     # For Resources
    SetDefaultSticky  BOOL,     # For Resources
    HasDocument       BOOL,     # For Resources
    ReadOnly          BOOL,     # For Resources
    TypeMap           JSON,

    PRIMARY KEY(SID),
    UNIQUE INDEX (RegistrySID, ParentSID, Plural),
    CONSTRAINT UC_Singular UNIQUE (RegistrySID, ParentSID, Singular)
);

CREATE TRIGGER ModelTrigger BEFORE DELETE ON ModelEntities
FOR EACH ROW
BEGIN
    DELETE FROM "Groups"        WHERE ModelSID=OLD.SID @
    DELETE FROM Resources       WHERE ModelSID=OLD.SID @
    DELETE FROM ModelAttributes WHERE ParentSID=OLD.SID @
END ;

# Not used yet
CREATE TABLE ModelAttributes (
    SID           VARCHAR(64) NOT NULL,   # my System ID
    RegistrySID   VARCHAR(64) NOT NULL,
    ParentSID     VARCHAR(64),            # NULL=Root. Model or IfValue SID
    Name          VARCHAR(64) NOT NULL,
    Type          VARCHAR(64) NOT NULL,
    Description   VARCHAR(255),
    Strict        BOOL NOT NULL,
    Required      BOOL NOT NULL,
    ItemType      VARCHAR(64),

    PRIMARY KEY(RegistrySID, ParentSID, SID),
    UNIQUE INDEX (SID),
    CONSTRAINT UC_Name UNIQUE (RegistrySID, ParentSID, Name)
);

CREATE TRIGGER ModelAttributeTrigger BEFORE DELETE ON ModelAttributes
FOR EACH ROW
BEGIN
    DELETE FROM ModelEnums    WHERE AttributeSID=OLD.SID @
    DELETE FROM ModelIfValues WHERE AttributeSID=OLD.SID @
END ;

CREATE TABLE ModelEnums (
    RegistrySID   VARCHAR(64) NOT NULL,
    AttributeSID  VARCHAR(64) NOT NULL,
    Value         VARCHAR(255) NOT NULL,

    PRIMARY KEY(RegistrySID, AttributeSID),
    INDEX (AttributeSID),
    CONSTRAINT UC_Value UNIQUE (RegistrySID, AttributeSID, Value)
);

CREATE TABLE ModelIfValues (
    SID           VARCHAR(64) NOT NULL,
    RegistrySID   VARCHAR(64) NOT NULL,
    AttributeSID  VARCHAR(64) NOT NULL,
    Value         VARCHAR(255) NOT NULL,

    PRIMARY KEY(RegistrySID, AttributeSID),
    UNIQUE INDEX (SID),
    INDEX (AttributeSID),
    CONSTRAINT UC_Value UNIQUE (RegistrySID, AttributeSID, Value)
);

CREATE TRIGGER ModelIfValuesTrigger BEFORE DELETE ON ModelIfValues
FOR EACH ROW
BEGIN
    DELETE FROM ModelAttributes    WHERE ParentSID=OLD.SID @
END ;


CREATE TABLE "Groups" (
    SID             VARCHAR(64) NOT NULL,   # System ID
    UID             VARCHAR(64) NOT NULL,   # User defined
    RegistrySID     VARCHAR(64) NOT NULL,
    ModelSID        VARCHAR(64) NOT NULL,
    Path            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract        VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,

    PRIMARY KEY (SID),
    INDEX(RegistrySID, UID),
    UNIQUE INDEX (RegistrySID, ModelSID, UID)
);

CREATE TRIGGER GroupTrigger BEFORE DELETE ON "Groups"
FOR EACH ROW
BEGIN
    DELETE FROM Props WHERE EntitySID=OLD.SID @
    DELETE FROM Resources WHERE GroupSID=OLD.SID @
END ;

CREATE TABLE Resources (
    SID             VARCHAR(64) NOT NULL,   # System ID
    UID             VARCHAR(64) NOT NULL,   # User defined
    GroupSID        VARCHAR(64) NOT NULL,   # System ID
    ModelSID        VARCHAR(64) NOT NULL,
    Path            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract        VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,

    PRIMARY KEY (SID),
    INDEX(GroupSID, UID),
    INDEX(Path),
    UNIQUE INDEX (GroupSID, ModelSID, UID)
);

CREATE TRIGGER ResourcesTrigger BEFORE DELETE ON Resources
FOR EACH ROW
BEGIN
    DELETE FROM Props WHERE EntitySID=OLD.SID @
    DELETE FROM Versions WHERE ResourceSID=OLD.SID @
END ;

CREATE TABLE Versions (
    SID                 VARCHAR(64) NOT NULL,   # System ID
    UID                 VARCHAR(64) NOT NULL,   # User defined, vID
    ResourceSID         VARCHAR(64) NOT NULL,   # System ID
    Path                VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Counter             SERIAL,                 # Counter, auto-increments

    ResourceURL         VARCHAR(255),
    ResourceProxyURL    VARCHAR(255),
    ResourceContentSID  VARCHAR(64),

    PRIMARY KEY (SID),
    UNIQUE INDEX (ResourceSID, UID),
    INDEX(ResourceSID)
);

CREATE TABLE Props (
    RegistrySID VARCHAR(64) NOT NULL,
    EntitySID   VARCHAR(64) NOT NULL,       # Reg,Group,Res,Ver System ID
    PropName    VARCHAR(64) NOT NULL,
    PropValue   VARCHAR(255),
    PropType    CHAR(64) NOT NULL,          # string, boolean, int, ...

    PRIMARY KEY (EntitySID, PropName),
    INDEX (EntitySID),
    INDEX (PropName)
);

CREATE VIEW xRefSrc2TgtResources AS
SELECT
    P.RegistrySID,
    P.EntitySID AS SourceSID,
    sR.Path AS SourcePath,
    sR.Abstract AS SourceAbstract,
    tR.SID as TargetSID,
    tR.Path as TargetPath
FROM Props AS P
JOIN Resources AS sR ON (sR.SID=P.EntitySID)
JOIN Resources AS tR ON (tR.Path=P.PropValue)
WHERE P.PropName='xref,' ;

CREATE VIEW xRefVersions AS
SELECT
    CONCAT(xR.SourceSID, '-', V.SID) AS SID,
    V.UID,
    xR.SourceSID AS ResourceSID,
    CONCAT(xR.SourcePath, '/versions/', V.UID) AS Path,
    CONCAT(xR.SourceAbstract, ',versions') AS Abstract,
    V.Counter,
    V.ResourceURL,
    V.ResourceProxyURL,
    V.ResourceContentSID
FROM xRefSrc2TgtResources AS xR
JOIN Versions AS V ON (V.ResourceSID=xR.TargetSID);

# This is Versions table + xref'd Versions
CREATE VIEW EffectiveVersions AS
SELECT * FROM Versions
UNION SELECT * FROM xRefVersions ;

CREATE TRIGGER VersionsTrigger BEFORE DELETE ON Versions
FOR EACH ROW
BEGIN
    DELETE FROM Props WHERE EntitySID=OLD.SID @
    DELETE FROM ResourceContents WHERE VersionSID=OLD.SID @
END ;

CREATE VIEW Entities AS
SELECT                          # Gather Registries
    r.SID AS RegSID,
    0 AS Level,
    'registries' AS Plural,
    NULL AS ParentSID,
    r.SID AS eSID,
    r.UID AS UID,
    '' AS Abstract,
    '' AS Path
FROM Registries AS r

UNION SELECT                            # Gather Groups
    g.RegistrySID AS RegSID,
    1 AS Level,
    m.Plural AS Plural,
    g.RegistrySID AS ParentSID,
    g.SID AS eSID,
    g.UID AS UID,
    g.Abstract,
    g.Path
FROM "Groups" AS g
JOIN ModelEntities AS m ON (m.SID=g.ModelSID)

UNION SELECT                    # Add Resources
    m.RegistrySID AS RegSID,
    2 AS Level,
    m.Plural AS Plural,
    r.GroupSID AS ParentSID,
    r.SID AS eSID,
    r.UID AS UID,
    r.Abstract,
    r.Path
FROM Resources AS r
JOIN ModelEntities AS m ON (m.SID=r.ModelSID)

UNION SELECT                    # Add Versions (including xref'd versions)
    rm.RegistrySID AS RegSID,
    3 AS Level,
    'versions' AS Plural,
    r.SID AS ParentSID,
    v.SID AS eSID,
    v.UID AS UID,
    v.Abstract,
    v.Path
FROM EffectiveVersions AS v
JOIN Resources AS r ON (r.SID=v.ResourceSID)
JOIN ModelEntities AS rm ON (rm.SID=r.ModelSID) ;

# Calculate the raw Props that need to be duplicated due to xRefs.
# This assumes other calculated props (like isDefault) will be done later
CREATE VIEW xRefProps AS
SELECT                            # Iterate over the xRef Resources
    R.RegistrySID,
    R.SourceSID AS EntitySID,
    P.PropName,
    P.PropValue,
    P.PropType
FROM xRefSrc2TgtResources AS R
JOIN Props AS P ON (              # Grab the Target Resource's attributes
  P.EntitySID=R.TargetSID AND
  P.PropName<>'id,' AND           # Don't override this
  P.PropName<>'xref,' )           # Don't override this
UNION SELECT                      # Grab the Target Version's attributes
    R.RegistrySID,
    CONCAT(R.SourceSID, '-', P.EntitySID) AS EntitySID,
    P.PropName,
    P.PropValue,
    P.PropType
FROM xRefSrc2TgtResources AS R
JOIN Props AS P ON (
  P.EntitySID IN (
    SELECT eSID FROM Entities WHERE ParentSID=R.TargetSID
  ) AND
  P.PropName<>'xref,' )           # Don't override this
;

# This is the Props table + xref'd props (for Resource and Versions)
CREATE VIEW EffectiveProps AS
SELECT * FROM Props
UNION SELECT * FROM xRefProps ;

CREATE TABLE ResourceContents (
    VersionSID      VARCHAR(255),
    Content         MEDIUMBLOB,

    PRIMARY KEY (VersionSID)
);

CREATE VIEW DefaultProps AS
SELECT
    p.RegistrySID,
    r.SID AS EntitySID,
    p.PropName,
    p.PropValue,
    p.PropType
FROM EffectiveProps AS p
JOIN EffectiveVersions AS v ON (p.EntitySID=v.SID)
JOIN Resources AS r ON (r.SID=v.ResourceSID)
JOIN EffectiveProps AS p1 ON (p1.EntitySID=r.SID)
WHERE p1.PropName='defaultVersionId,' AND v.UID=p1.PropValue AND
      p.PropName<>'id,' ;     # Don't overwrite this
# NOTE!!! if DB_IN changes then the above 2 lines MUST change
# TODO move the creation of this into the code then we can dynamically
# use DB_IN instead of hard-coding the "," in here

CREATE VIEW AllProps AS
SELECT * FROM EffectiveProps
UNION SELECT * FROM DefaultProps
UNION SELECT                    # Add in "isdefault", which is calculated
  v.RegSID,
  v.eSID,
  'isdefault,',
  'true',
  'boolean'
FROM Entities AS v
JOIN EffectiveProps AS p ON (
  p.EntitySID=v.ParentSID AND
  p.PropName='defaultversionid,'
  AND p.PropValue=v.UID );

CREATE VIEW xRefResources AS
SELECT
    xR.SourceSID AS SID,
    R.UID,
    R.GroupSID,
    R.ModelSID,
    R.Path,
    R.Abstract
FROM xRefSrc2TgtResources AS xR
JOIN Resources AS R ON (R.SID=xR.SourceSID) ;

CREATE VIEW FullTree AS
SELECT
    RegSID,
    Level,
    Plural,
    ParentSID,
    eSID,
    UID,
    Path,
    PropName,
    PropValue,
    PropType,
    Abstract
FROM Entities
LEFT JOIN AllProps ON (AllProps.EntitySID=Entities.eSID)
ORDER by Path, PropName;

CREATE VIEW Leaves AS
SELECT eSID FROM Entities
WHERE eSID NOT IN (
    SELECT DISTINCT ParentSID FROM Entities WHERE ParentSID IS NOT NULL
);

