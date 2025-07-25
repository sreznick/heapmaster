package hprof

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

// InitDB opens connection and migrates schema
func InitDB() error {
	dsn := "host=127.0.0.1 user=user password=password dbname=postgres port=15432 sslmode=disable TimeZone=UTC"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get generic database object: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Auto migrate all tables
	tables := []interface{}{
		&StringInUTF8{},
		&LoadClass{},
		&UnloadClass{},
		&StackTrace{},
		&StackFrame{},
		&AllocSites{},
		&Site{},
		&RootUnknown{},
		&RootJNIGlobal{},
		&RootJNILocal{},
		&RootJavaFrame{},
		&RootNativeStack{},
		&RootStickyClass{},
		&RootThreadBlock{},
		&RootMonitorUsed{},
		&RootThreadObject{},
		&ClassDump{},
		&ConstantPoolRecord{},
		&StaticFieldRecord{},
		&InstanceFieldRecord{},
		&InstanceDump{},
		&InstanceFieldValues{},
		&ObjectArrayDump{},
		&ObjectArrayElement{},
		&PrimitiveArrayDump{},
		&PrimitiveArrayElement{},
	}

	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			return fmt.Errorf("failed to migrate table %T: %w", table, err)
		}
	}

	log.Println("Database schema migrated successfully!")
	return nil
}

func GetDB() *gorm.DB {
	return db
}

func IsDBInitialized() bool {
	return db != nil
}

func SaveStringInUTF8(s *StringInUTF8) error {
	return db.Create(s).Error
}

func SaveLoadClass(lc *LoadClass) error {
	return db.Create(lc).Error
}

func SaveUnloadClass(uc *UnloadClass) error {
	return db.Create(uc).Error
}

func SaveStackTrace(st *StackTrace) error {
	return db.Create(st).Error
}

func SaveStackFrame(sf *StackFrame) error {
	return db.Create(sf).Error
}

func SaveAllocSites(as *AllocSites) error {
	return db.Create(as).Error
}

func SaveSite(s *Site) error {
	return db.Create(s).Error
}

func SaveRootUnknown(ru *RootUnknown) error {
	return db.Create(ru).Error
}

func SaveRootJNIGlobal(rj *RootJNIGlobal) error {
	return db.Create(rj).Error
}

func SaveRootJNILocal(rl *RootJNILocal) error {
	return db.Create(rl).Error
}

func SaveRootJavaFrame(rj *RootJavaFrame) error {
	return db.Create(rj).Error
}

func SaveRootNativeStack(rn *RootNativeStack) error {
	return db.Create(rn).Error
}

func SaveRootStickyClass(rs *RootStickyClass) error {
	return db.Create(rs).Error
}

func SaveRootThreadBlock(rt *RootThreadBlock) error {
	return db.Create(rt).Error
}

func SaveRootMonitorUsed(rm *RootMonitorUsed) error {
	return db.Create(rm).Error
}

func SaveRootThreadObject(rt *RootThreadObject) error {
	return db.Create(rt).Error
}

func SaveClassDump(cd *ClassDump) error {
	return db.Create(cd).Error
}

func SaveConstantPoolRecord(cpr *ConstantPoolRecord) error {
	return db.Create(cpr).Error
}

func SaveStaticFieldRecord(sfr *StaticFieldRecord) error {
	return db.Create(sfr).Error
}

func SaveInstanceFieldRecord(ifr *InstanceFieldRecord) error {
	return db.Create(ifr).Error
}

func SaveInstanceDump(id *InstanceDump) error {
	return db.Create(id).Error
}

func SaveInstanceFieldValues(ifv *InstanceFieldValues) error {
	return db.Create(ifv).Error
}

func SaveObjectArrayDump(oad *ObjectArrayDump) error {
	return db.Create(oad).Error
}

func SaveObjectArrayElement(oae *ObjectArrayElement) error {
	return db.Create(oae).Error
}

func SavePrimitiveArrayDump(pad *PrimitiveArrayDump) error {
	return db.Create(pad).Error
}

func SavePrimitiveArrayElement(pae *PrimitiveArrayElement) error {
	return db.Create(pae).Error
}
