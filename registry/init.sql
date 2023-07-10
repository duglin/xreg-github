SET GLOBAL sql_mode = 'ANSI_QUOTES' ;

DROP DATABASE IF EXISTS registry ;
CREATE DATABASE registry ;
USE registry ;

CREATE TABLE Registries (
	ID          VARCHAR(255) NOT NULL,	# System ID
	RegistryID	VARCHAR(255) NOT NULL,	# User defined

	PRIMARY KEY (ID)
);

CREATE TRIGGER RegistryTrigger BEFORE DELETE ON Registries
FOR EACH ROW
BEGIN
  DELETE FROM Props WHERE EntityID=OLD.ID @
  DELETE FROM "Groups" WHERE RegistryID=OLD.ID @
  DELETE FROM ModelEntities WHERE RegistryID=OLD.ID @
END ;

CREATE TABLE ModelEntities (		# Group or Resource (no parent->Group)
	ID     		VARCHAR(64),		# my System ID
	RegistryID  VARCHAR(64),
	ParentID	VARCHAR(64),		# ID of parent ModelEntity

	Plural		VARCHAR(64),
	Singular	VARCHAR(64),
	SchemaURL	VARCHAR(255),		# For Groups
	Versions    INT NOT NULL,		# For Resources
	VersionId   BOOL NOT NULL,		# For Resources
	Latest      BOOL NOT NULL,		# For Resources

	PRIMARY KEY(ID),
	INDEX (RegistryID, ParentID, Plural)
);

CREATE TRIGGER ModelTrigger BEFORE DELETE ON ModelEntities
FOR EACH ROW
BEGIN
  DELETE FROM "Groups" WHERE ModelID=OLD.ID @
  DELETE FROM Resources WHERE ModelID=OLD.ID @
END ;


CREATE TABLE "Groups" (
	ID				VARCHAR(64) NOT NULL,	# System ID
	RegistryID		VARCHAR(64) NOT NULL,
	GroupID			VARCHAR(64) NOT NULL,	# User defined
	ModelID			VARCHAR(64) NOT NULL,
	Path			VARCHAR(255) NOT NULL,
	Abstract		VARCHAR(255) NOT NULL,

	PRIMARY KEY (ID),
	INDEX(GroupID)
);

CREATE TRIGGER GroupTrigger BEFORE DELETE ON "Groups"
FOR EACH ROW
BEGIN
  DELETE FROM Props WHERE EntityID=OLD.ID @
  DELETE FROM Resources WHERE GroupID=OLD.ID @
END ;

CREATE TABLE Resources (
	ID				VARCHAR(64) NOT NULL,	# System ID
	ResourceID      VARCHAR(64) NOT NULL,	# User defined
	GroupID			VARCHAR(64) NOT NULL,	# System ID
	ModelID         VARCHAR(64) NOT NULL,
	Path			VARCHAR(255) NOT NULL,
	Abstract		VARCHAR(255) NOT NULL,

	PRIMARY KEY (ID),
	INDEX(ResourceID)
);

CREATE TRIGGER ResourcesTrigger BEFORE DELETE ON Resources
FOR EACH ROW
BEGIN
  DELETE FROM Props WHERE EntityID=OLD.ID @
  DELETE FROM Versions WHERE ResourceID=OLD.ID @
END ;

CREATE TABLE Versions (
	ID					VARCHAR(64) NOT NULL,	# System ID
	VersionID			VARCHAR(64) NOT NULL,	# User defined
	ResourceID			VARCHAR(64) NOT NULL,	# System ID
	Path				VARCHAR(255) NOT NULL,
	Abstract			VARCHAR(255) NOT NULL,

	ResourceURL     	VARCHAR(255),
	ResourceProxyURL	VARCHAR(255),
	ResourceContentID	VARCHAR(64),

	PRIMARY KEY (ID),
	INDEX (VersionID)
);

CREATE TRIGGER VersionsTrigger BEFORE DELETE ON Versions
FOR EACH ROW
BEGIN
  DELETE FROM Props WHERE EntityID=OLD.ID @
  DELETE FROM ResourceContents WHERE VersionID=OLD.ID @
END ;

CREATE TABLE Props (
	RegistryID  VARCHAR(64) NOT NULL,
	EntityID	VARCHAR(64) NOT NULL,		# Reg,Group,Res,Ver System ID
	PropName	VARCHAR(64) NOT NULL,
	PropValue	VARCHAR(255),
	PropType	VARCHAR(64) NOT NULL,

	PRIMARY KEY (EntityID, PropName),
	INDEX (EntityID)
);

CREATE TABLE ResourceContents (
	VersionID		VARCHAR(255),
	Content			MEDIUMBLOB,

	PRIMARY KEY (VersionID)
);

CREATE VIEW LatestProps AS
SELECT
	p.RegistryID,
	r.ID AS EntityID,
	p.PropName,
	p.PropValue,
	p.PropType
FROM Props AS p
JOIN Versions AS v ON (p.EntityID=v.ID)
JOIN Resources AS r ON (r.ID=v.ResourceID)
JOIN Props AS p1 ON (p1.EntityID=r.ID)
WHERE p1.PropName='LatestId' AND v.VersionID=p1.PropValue AND
	  p.PropName<>'id';		# Don't overwrite 'id'

CREATE VIEW AllProps AS
SELECT * FROM Props
UNION SELECT * FROM LatestProps ;


CREATE VIEW Entities AS
SELECT							# Gather Registries
	r.ID AS RegID,
	0 AS Level,
	'registries' AS Plural,
	NULL AS ParentID,
	r.ID AS eID,
	r.RegistryID AS ID,
	'' AS Abstract,
	'' AS Path
FROM Registries AS r

UNION SELECT							# Gather Groups
	g.RegistryID AS RegID,
	1 AS Level,
	m.Plural AS Plural,
	g.RegistryID AS ParentID,
	g.ID AS eID,
	g.GroupID AS ID,
	g.Abstract,
	g.Path
FROM "Groups" AS g
JOIN ModelEntities AS m ON (m.ID=g.ModelID)

UNION SELECT					# Add Resources
	m.RegistryID AS RegID,
	2 AS Level,
	m.Plural AS Plural,
	r.GroupID AS ParentID,
	r.ID AS eID,
	r.ResourceID AS ID,
	r.Abstract,
	r.Path
FROM Resources AS r
JOIN ModelEntities AS m ON (m.ID=r.ModelID)

UNION SELECT					# Add Versions
	rm.RegistryID AS RegID,
	3 AS Level,
	'versions' AS Plural,
	r.ID AS ParentID,
	v.ID AS eID,
	v.VersionID AS ID,
	v.Abstract,
	v.Path
FROM Versions AS v
JOIN Resources AS r ON (r.ID=v.ResourceID)
JOIN ModelEntities AS rm ON (rm.ID=r.ModelID) ;

CREATE VIEW FullTree AS
SELECT
	RegID,
	Level,
	Plural,
	ParentID,
	eID,
	ID,
	Path,
	PropName,
	PropValue,
	PropType,
	Abstract
FROM Entities
LEFT JOIN AllProps ON (AllProps.EntityID=Entities.eID)
ORDER by Path, PropName;

CREATE VIEW Leaves AS
SELECT eID FROM Entities
WHERE eID NOT IN (
	SELECT DISTINCT ParentID FROM Entities WHERE ParentID IS NOT NULL
);

