package exif

import (
	"bytes"
	"fmt"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/dsoprea/go-logging"
)

func TestIfdBuilder_Add(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	originalBytes := []byte{0x11, 0x22, 0x33}

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x44,
		value:   NewIfdBuilderTagValueFromBytes([]byte(originalBytes)),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	if ib.ifdPath != IfdPathStandard {
		t.Fatalf("IFD name not correct.")
	} else if ib.ifdTagId != 0 {
		t.Fatalf("IFD tag-ID not correct.")
	} else if ib.byteOrder != TestDefaultByteOrder {
		t.Fatalf("IFD byte-order not correct.")
	} else if len(ib.tags) != 4 {
		t.Fatalf("IFD tag-count not correct.")
	} else if ib.existingOffset != 0 {
		t.Fatalf("IFD offset not correct.")
	} else if ib.nextIb != nil {
		t.Fatalf("Next-IFD not correct.")
	}

	tags := ib.Tags()

	if tags[0].tagId != 0x11 {
		t.Fatalf("tag (0) tag-ID not correct")
	} else if bytes.Compare(tags[0].value.Bytes(), []byte("test string")) != 0 {
		t.Fatalf("tag (0) value not correct")
	}

	if tags[1].tagId != 0x22 {
		t.Fatalf("tag (1) tag-ID not correct")
	} else if bytes.Compare(tags[1].value.Bytes(), []byte("test string2")) != 0 {
		t.Fatalf("tag (1) value not correct")
	}

	if tags[2].tagId != 0x33 {
		t.Fatalf("tag (2) tag-ID not correct")
	} else if bytes.Compare(tags[2].value.Bytes(), []byte("test string3")) != 0 {
		t.Fatalf("tag (2) value not correct")
	}

	if tags[3].tagId != 0x44 {
		t.Fatalf("tag (3) tag-ID not correct")
	} else if bytes.Compare(tags[3].value.Bytes(), originalBytes) != 0 {
		t.Fatalf("tag (3) value not correct")
	}
}

func TestIfdBuilder_SetNextIb(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()

	ib1 := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)
	ib2 := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	if ib1.nextIb != nil {
		t.Fatalf("Next-IFD for IB1 not initially terminal.")
	}

	err = ib1.SetNextIb(ib2)
	log.PanicIf(err)

	if ib1.nextIb != ib2 {
		t.Fatalf("Next-IFD for IB1 not correct.")
	} else if ib2.nextIb != nil {
		t.Fatalf("Next-IFD for IB2 terminal.")
	}
}

func TestIfdBuilder_AddChildIb(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	ibChild := NewIfdBuilder(im, ti, IfdPathStandardExif, TestDefaultByteOrder)
	err = ib.AddChildIb(ibChild)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	if ib.tags[0].tagId != 0x11 {
		t.Fatalf("first tag not correct")
	} else if ib.tags[1].tagId != ibChild.ifdTagId {
		t.Fatalf("second tag ID does not match child-IFD tag-ID: (0x%04x) != (0x%04x)", ib.tags[1].tagId, ibChild.ifdTagId)
	} else if ib.tags[1].value.Ib() != ibChild {
		t.Fatalf("second tagvalue does not match child-IFD")
	} else if ib.tags[2].tagId != 0x22 {
		t.Fatalf("third tag not correct")
	}
}

func TestIfdBuilder_AddTagsFromExisting(t *testing.T) {
	defer func() {
		if state := recover(); state != nil {
			err := log.Wrap(state.(error))
			log.PrintError(err)

			t.Fatalf("Test failure.")
		}
	}()

	exifData := getExifSimpleTestIbBytes()

	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()

	_, index, err := Collect(im, ti, exifData)
	log.PanicIf(err)

	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	err = ib.AddTagsFromExisting(index.RootIfd, nil, nil, nil)
	log.PanicIf(err)

	expected := []uint16{
		0x000b,
		0x00ff,
		0x0100,
		0x013e,
	}

	if len(ib.tags) != len(expected) {
		t.Fatalf("Tag count not correct: (%d) != (%d)", len(ib.tags), len(expected))
	}

	for i, tag := range ib.tags {
		if tag.tagId != expected[i] {
			t.Fatalf("Tag (%d) not correct: (0x%04x) != (0x%04x)", i, tag.tagId, expected[i])
		}
	}
}

func TestIfdBuilder_AddTagsFromExisting__Includes(t *testing.T) {
	exifData := getExifSimpleTestIbBytes()

	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()

	_, index, err := Collect(im, ti, exifData)
	log.PanicIf(err)

	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	err = ib.AddTagsFromExisting(index.RootIfd, nil, []uint16{0x00ff}, nil)
	log.PanicIf(err)

	expected := []uint16{
		0x00ff,
	}

	if len(ib.tags) != len(expected) {
		t.Fatalf("Tag count not correct: (%d) != (%d)", len(ib.tags), len(expected))
	}

	for i, tag := range ib.tags {
		if tag.tagId != expected[i] {
			t.Fatalf("Tag (%d) not correct: (0x%04x) != (0x%04x)", i, tag.tagId, expected[i])
		}
	}
}

func TestIfdBuilder_AddTagsFromExisting__Excludes(t *testing.T) {
	exifData := getExifSimpleTestIbBytes()

	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()

	_, index, err := Collect(im, ti, exifData)
	log.PanicIf(err)

	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	err = ib.AddTagsFromExisting(index.RootIfd, nil, nil, []uint16{0xff})
	log.PanicIf(err)

	expected := []uint16{
		0x000b,
		0x0100,
		0x013e,
	}

	if len(ib.tags) != len(expected) {
		t.Fatalf("Tag count not correct: (%d) != (%d)", len(ib.tags), len(expected))
	}

	for i, tag := range ib.tags {
		if tag.tagId != expected[i] {
			t.Fatalf("Tag (%d) not correct: (0x%04x) != (0x%04x)", i, tag.tagId, expected[i])
		}
	}
}

func TestIfdBuilder_FindN__First_1(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	found, err := ib.FindN(0x11, 1)
	log.PanicIf(err)

	if len(found) != 1 {
		log.Panicf("Exactly one result was not found: (%d)", len(found))
	} else if found[0] != 0 {
		log.Panicf("Result was not in the right place: (%d)", found[0])
	}

	tags := ib.Tags()
	bt = tags[found[0]]

	if bt.tagId != 0x11 {
		log.Panicf("Found entry is not correct: (0x%04x)", bt.tagId)
	}
}

func TestIfdBuilder_FindN__First_2_1Returned(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	found, err := ib.FindN(0x11, 2)
	log.PanicIf(err)

	if len(found) != 1 {
		log.Panicf("Exactly one result was not found: (%d)", len(found))
	} else if found[0] != 0 {
		log.Panicf("Result was not in the right place: (%d)", found[0])
	}

	tags := ib.Tags()
	bt = tags[found[0]]

	if bt.tagId != 0x11 {
		log.Panicf("Found entry is not correct: (0x%04x)", bt.tagId)
	}
}

func TestIfdBuilder_FindN__First_2_2Returned(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string5")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	found, err := ib.FindN(0x11, 2)
	log.PanicIf(err)

	if len(found) != 2 {
		log.Panicf("Exactly one result was not found: (%d)", len(found))
	} else if found[0] != 0 {
		log.Panicf("First result was not in the right place: (%d)", found[0])
	} else if found[1] != 3 {
		log.Panicf("Second result was not in the right place: (%d)", found[1])
	}

	tags := ib.Tags()

	bt = tags[found[0]]
	if bt.tagId != 0x11 || bytes.Compare(bt.value.Bytes(), []byte("test string")) != 0 {
		log.Panicf("Found entry 0 is not correct: (0x%04x) [%s]", bt.tagId, bt.value)
	}

	bt = tags[found[1]]
	if bt.tagId != 0x11 || bytes.Compare(bt.value.Bytes(), []byte("test string4")) != 0 {
		log.Panicf("Found entry 1 is not correct: (0x%04x) [%s]", bt.tagId, bt.value)
	}
}

func TestIfdBuilder_FindN__Middle_WithDuplicates(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string5")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string6")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	found, err := ib.FindN(0x33, 1)
	log.PanicIf(err)

	if len(found) != 1 {
		log.Panicf("Exactly one result was not found: (%d)", len(found))
	} else if found[0] != 2 {
		log.Panicf("Result was not in the right place: (%d)", found[0])
	}

	tags := ib.Tags()
	bt = tags[found[0]]

	if bt.tagId != 0x33 {
		log.Panicf("Found entry is not correct: (0x%04x)", bt.tagId)
	}
}

func TestIfdBuilder_FindN__Middle_NoDuplicates(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	found, err := ib.FindN(0x33, 1)
	log.PanicIf(err)

	if len(found) != 1 {
		log.Panicf("Exactly one result was not found: (%d)", len(found))
	} else if found[0] != 2 {
		log.Panicf("Result was not in the right place: (%d)", found[0])
	}

	tags := ib.Tags()
	bt = tags[found[0]]

	if bt.tagId != 0x33 {
		log.Panicf("Found entry is not correct: (0x%04x)", bt.tagId)
	}
}

func TestIfdBuilder_FindN__Miss(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	found, err := ib.FindN(0x11, 1)
	log.PanicIf(err)

	if len(found) != 0 {
		t.Fatalf("Expected empty results.")
	}
}

func TestIfdBuilder_Find__Hit(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	position, err := ib.Find(0x33)
	log.PanicIf(err)

	if position != 2 {
		log.Panicf("Result was not in the right place: (%d)", position)
	}

	tags := ib.Tags()
	bt = tags[position]

	if bt.tagId != 0x33 {
		log.Panicf("Found entry is not correct: (0x%04x)", bt.tagId)
	}
}

func TestIfdBuilder_Find__Miss(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	_, err = ib.Find(0x99)
	if err == nil {
		t.Fatalf("Expected an error.")
	} else if log.Is(err, ErrTagEntryNotFound) == false {
		log.Panic(err)
	}
}

func TestIfdBuilder_Replace(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	currentIds := make([]uint16, 3)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x22, 0x33}, currentIds) == false {
		t.Fatalf("Pre-replace tags are not correct.")
	}

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x99,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.Replace(0x22, bt)
	log.PanicIf(err)

	currentIds = make([]uint16, 3)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x99, 0x33}, currentIds) == false {
		t.Fatalf("Post-replace tags are not correct.")
	}
}

func TestIfdBuilder_ReplaceN(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	currentIds := make([]uint16, 3)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x22, 0x33}, currentIds) == false {
		t.Fatalf("Pre-replace tags are not correct.")
	}

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0xA9,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.ReplaceAt(1, bt)
	log.PanicIf(err)

	currentIds = make([]uint16, 3)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0xA9, 0x33}, currentIds) == false {
		t.Fatalf("Post-replace tags are not correct.")
	}
}

func TestIfdBuilder_DeleteFirst(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	if len(ib.Tags()) != 4 {
		t.Fatalf("Pre-delete tag count not correct.")
	}

	currentIds := make([]uint16, 4)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x22, 0x22, 0x33}, currentIds) == false {
		t.Fatalf("Pre-delete tags not correct.")
	}

	err = ib.DeleteFirst(0x22)
	log.PanicIf(err)

	if len(ib.Tags()) != 3 {
		t.Fatalf("Post-delete (1) tag count not correct.")
	}

	currentIds = make([]uint16, 3)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x22, 0x33}, currentIds) == false {
		t.Fatalf("Post-delete (1) tags not correct.")
	}

	err = ib.DeleteFirst(0x22)
	log.PanicIf(err)

	if len(ib.Tags()) != 2 {
		t.Fatalf("Post-delete (2) tag count not correct.")
	}

	currentIds = make([]uint16, 2)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x33}, currentIds) == false {
		t.Fatalf("Post-delete (2) tags not correct.")
	}

	err = ib.DeleteFirst(0x22)
	if err == nil {
		t.Fatalf("Expected an error.")
	} else if log.Is(err, ErrTagEntryNotFound) == false {
		log.Panic(err)
	}
}

func TestIfdBuilder_DeleteN(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	if len(ib.Tags()) != 4 {
		t.Fatalf("Pre-delete tag count not correct.")
	}

	currentIds := make([]uint16, 4)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x22, 0x22, 0x33}, currentIds) == false {
		t.Fatalf("Pre-delete tags not correct.")
	}

	err = ib.DeleteN(0x22, 1)
	log.PanicIf(err)

	if len(ib.Tags()) != 3 {
		t.Fatalf("Post-delete (1) tag count not correct.")
	}

	currentIds = make([]uint16, 3)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x22, 0x33}, currentIds) == false {
		t.Fatalf("Post-delete (1) tags not correct.")
	}

	err = ib.DeleteN(0x22, 1)
	log.PanicIf(err)

	if len(ib.Tags()) != 2 {
		t.Fatalf("Post-delete (2) tag count not correct.")
	}

	currentIds = make([]uint16, 2)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x33}, currentIds) == false {
		t.Fatalf("Post-delete (2) tags not correct.")
	}

	err = ib.DeleteN(0x22, 1)
	if err == nil {
		t.Fatalf("Expected an error.")
	} else if log.Is(err, ErrTagEntryNotFound) == false {
		log.Panic(err)
	}
}

func TestIfdBuilder_DeleteN_Two(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	if len(ib.Tags()) != 4 {
		t.Fatalf("Pre-delete tag count not correct.")
	}

	currentIds := make([]uint16, 4)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x22, 0x22, 0x33}, currentIds) == false {
		t.Fatalf("Pre-delete tags not correct.")
	}

	err = ib.DeleteN(0x22, 2)
	log.PanicIf(err)

	if len(ib.Tags()) != 2 {
		t.Fatalf("Post-delete tag count not correct.")
	}

	currentIds = make([]uint16, 2)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x33}, currentIds) == false {
		t.Fatalf("Post-delete tags not correct.")
	}

	err = ib.DeleteFirst(0x22)
	if err == nil {
		t.Fatalf("Expected an error.")
	} else if log.Is(err, ErrTagEntryNotFound) == false {
		log.Panic(err)
	}
}

func TestIfdBuilder_DeleteAll(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	bt := &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x11,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string2")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x22,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string3")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	bt = &BuilderTag{
		ifdPath: IfdPathStandard,
		typeId:  TypeByte,
		tagId:   0x33,
		value:   NewIfdBuilderTagValueFromBytes([]byte("test string4")),
	}

	err = ib.Add(bt)
	log.PanicIf(err)

	if len(ib.Tags()) != 4 {
		t.Fatalf("Pre-delete tag count not correct.")
	}

	currentIds := make([]uint16, 4)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x22, 0x22, 0x33}, currentIds) == false {
		t.Fatalf("Pre-delete tags not correct.")
	}

	n, err := ib.DeleteAll(0x22)
	log.PanicIf(err)

	if n != 2 {
		t.Fatalf("Returned delete tag count not correct.")
	} else if len(ib.Tags()) != 2 {
		t.Fatalf("Post-delete tag count not correct.")
	}

	currentIds = make([]uint16, 2)
	for i, bt := range ib.Tags() {
		currentIds[i] = bt.tagId
	}

	if reflect.DeepEqual([]uint16{0x11, 0x33}, currentIds) == false {
		t.Fatalf("Post-delete tags not correct.")
	}

	err = ib.DeleteFirst(0x22)
	if err == nil {
		t.Fatalf("Expected an error.")
	} else if log.Is(err, ErrTagEntryNotFound) == false {
		log.Panic(err)
	}
}

func TestIfdBuilder_CreateIfdBuilderFromExistingChain(t *testing.T) {
	defer func() {
		if state := recover(); state != nil {
			err := log.Wrap(state.(error))
			log.PrintErrorf(err, "Test failure.")
		}
	}()

	filepath := path.Join(assetsPath, "NDM_8901.jpg")

	rawExif, err := SearchFileAndExtractExif(filepath)
	log.PanicIf(err)

	im := NewIfdMapping()

	err = LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()

	_, index, err := Collect(im, ti, rawExif)
	log.PanicIf(err)

	ib := NewIfdBuilderFromExistingChain(index.RootIfd, nil)

	actual := ib.DumpToStrings()

	expected := []string{
		"IFD<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-INDEX=(0) IFD-TAG-ID=(0x0000) TAG=[0x0000]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(0) TAG=[0x010f]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(1) TAG=[0x0110]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(2) TAG=[0x0112]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(3) TAG=[0x011a]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(4) TAG=[0x011b]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(5) TAG=[0x0128]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(6) TAG=[0x0132]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(7) TAG=[0x013b]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(8) TAG=[0x0213]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(9) TAG=[0x8298]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[IFD/Exif] TAG-INDEX=(10) TAG=[0x8769]>",
		"IFD<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-INDEX=(0) IFD-TAG-ID=(0x8769) TAG=[0x8769]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(0) TAG=[0x829a]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(1) TAG=[0x829d]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(2) TAG=[0x8822]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(3) TAG=[0x8827]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(4) TAG=[0x8830]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(5) TAG=[0x8832]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(6) TAG=[0x9000]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(7) TAG=[0x9003]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(8) TAG=[0x9004]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(9) TAG=[0x9101]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(10) TAG=[0x9201]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(11) TAG=[0x9202]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(12) TAG=[0x9204]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(13) TAG=[0x9207]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(14) TAG=[0x9209]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(15) TAG=[0x920a]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(16) TAG=[0x927c]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(17) TAG=[0x9286]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(18) TAG=[0x9290]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(19) TAG=[0x9291]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(20) TAG=[0x9292]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(21) TAG=[0xa000]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(22) TAG=[0xa001]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(23) TAG=[0xa002]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(24) TAG=[0xa003]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[IFD/Exif/Iop] TAG-INDEX=(25) TAG=[0xa005]>",
		"IFD<PARENTS=[IFD->IFD/Exif] FQ-IFD-PATH=[IFD/Exif/Iop] IFD-INDEX=(0) IFD-TAG-ID=(0xa005) TAG=[0xa005]>",
		"TAG<PARENTS=[IFD->IFD/Exif] FQ-IFD-PATH=[IFD/Exif/Iop] IFD-TAG-ID=(0xa005) CHILD-IFD=[] TAG-INDEX=(0) TAG=[0x0001]>",
		"TAG<PARENTS=[IFD->IFD/Exif] FQ-IFD-PATH=[IFD/Exif/Iop] IFD-TAG-ID=(0xa005) CHILD-IFD=[] TAG-INDEX=(1) TAG=[0x0002]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(26) TAG=[0xa20e]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(27) TAG=[0xa20f]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(28) TAG=[0xa210]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(29) TAG=[0xa401]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(30) TAG=[0xa402]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(31) TAG=[0xa403]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(32) TAG=[0xa406]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(33) TAG=[0xa430]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(34) TAG=[0xa431]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(35) TAG=[0xa432]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(36) TAG=[0xa434]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[] TAG-INDEX=(37) TAG=[0xa435]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[IFD/GPSInfo] TAG-INDEX=(11) TAG=[0x8825]>",
		"IFD<PARENTS=[IFD] FQ-IFD-PATH=[IFD/GPSInfo] IFD-INDEX=(0) IFD-TAG-ID=(0x8825) TAG=[0x8825]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/GPSInfo] IFD-TAG-ID=(0x8825) CHILD-IFD=[] TAG-INDEX=(0) TAG=[0x0000]>",
		"IFD<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-INDEX=(1) IFD-TAG-ID=(0x0000) TAG=[0x0000]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(0) TAG=[0x0201]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(1) TAG=[0x0202]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(2) TAG=[0x0103]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(3) TAG=[0x011a]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(4) TAG=[0x011b]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[] TAG-INDEX=(5) TAG=[0x0128]>",
	}

	if reflect.DeepEqual(actual, expected) == false {
		fmt.Printf("ACTUAL:\n%s\n\nEXPECTED:\n%s\n", strings.Join(actual, "\n"), strings.Join(expected, "\n"))
		t.Fatalf("IB did not [correctly] duplicate the IFD structure.")
	}
}

// TODO(dustin): !! Test with an actual GPS-attached image.

func TestIfdBuilder_CreateIfdBuilderFromExistingChain_RealData(t *testing.T) {
	filepath := path.Join(assetsPath, "NDM_8901.jpg")

	rawExif, err := SearchFileAndExtractExif(filepath)
	log.PanicIf(err)

	// Decode from binary.

	im := NewIfdMapping()

	err = LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()

	_, originalIndex, err := Collect(im, ti, rawExif)
	log.PanicIf(err)

	originalThumbnailData, err := originalIndex.RootIfd.NextIfd.Thumbnail()
	log.PanicIf(err)

	originalTags := originalIndex.RootIfd.DumpTags()

	// Encode back to binary.

	ibe := NewIfdByteEncoder()

	rootIb := NewIfdBuilderFromExistingChain(originalIndex.RootIfd, nil)

	updatedExif, err := ibe.EncodeToExif(rootIb)
	log.PanicIf(err)

	// Parse again.

	_, recoveredIndex, err := Collect(im, ti, updatedExif)
	log.PanicIf(err)

	recoveredTags := recoveredIndex.RootIfd.DumpTags()

	recoveredThumbnailData, err := recoveredIndex.RootIfd.NextIfd.Thumbnail()
	log.PanicIf(err)

	// Check the thumbnail.

	if bytes.Compare(recoveredThumbnailData, originalThumbnailData) != 0 {
		t.Fatalf("recovered thumbnail does not match original")
	}

	// Validate that all of the same IFDs were presented.

	originalIfdTags := make([][2]interface{}, 0)
	for _, ite := range originalTags {
		if ite.ChildIfdPath != "" {
			originalIfdTags = append(originalIfdTags, [2]interface{}{ite.IfdPath, ite.TagId})
		}
	}

	recoveredIfdTags := make([][2]interface{}, 0)
	for _, ite := range recoveredTags {
		if ite.ChildIfdPath != "" {
			recoveredIfdTags = append(recoveredIfdTags, [2]interface{}{ite.IfdPath, ite.TagId})
		}
	}

	if reflect.DeepEqual(recoveredIfdTags, originalIfdTags) != true {
		fmt.Printf("Original IFD tags:\n\n")

		for i, x := range originalIfdTags {
			fmt.Printf("  %02d %v\n", i, x)
		}

		fmt.Printf("\nRecovered IFD tags:\n\n")

		for i, x := range recoveredIfdTags {
			fmt.Printf("  %02d %v\n", i, x)
		}

		fmt.Printf("\n")

		t.Fatalf("Recovered IFD tags are not correct.")
	}

	// Validate that all of the tags owned by the IFDs were presented. Note
	// that the thumbnail tags are not kept but only produced on the fly, which
	// is why we check it above.

	if len(recoveredTags) != len(originalTags) {
		t.Fatalf("Recovered tag-count does not match original.")
	}

	for i, recoveredIte := range recoveredTags {
		if recoveredIte.ChildIfdPath != "" {
			continue
		}

		originalIte := originalTags[i]

		if recoveredIte.IfdPath != originalIte.IfdPath {
			t.Fatalf("IfdIdentity not as expected: %s != %s  ITE=%s", recoveredIte.IfdPath, originalIte.IfdPath, recoveredIte)
		} else if recoveredIte.TagId != originalIte.TagId {
			t.Fatalf("Tag-ID not as expected: %d != %d  ITE=%s", recoveredIte.TagId, originalIte.TagId, recoveredIte)
		} else if recoveredIte.TagType != originalIte.TagType {
			t.Fatalf("Tag-type not as expected: %d != %d  ITE=%s", recoveredIte.TagType, originalIte.TagType, recoveredIte)
		}

		// TODO(dustin): We're always accessing the addressable-data using the root-IFD. It shouldn't matter, but we'd rather access it from our specific IFD.
		originalValueBytes, err := originalIte.ValueBytes(originalIndex.RootIfd.addressableData, originalIndex.RootIfd.ByteOrder)
		log.PanicIf(err)

		recoveredValueBytes, err := recoveredIte.ValueBytes(recoveredIndex.RootIfd.addressableData, recoveredIndex.RootIfd.ByteOrder)
		log.PanicIf(err)

		if bytes.Compare(originalValueBytes, recoveredValueBytes) != 0 {
			t.Fatalf("bytes of tag content not correct: %s != %s", originalIte, recoveredIte)
		}
	}
}

// func TestIfdBuilder_CreateIfdBuilderFromExistingChain_RealData_WithUpdate(t *testing.T) {
// 	filepath := path.Join(assetsPath, "NDM_8901.jpg")

// 	rawExif, err := SearchFileAndExtractExif(filepath)
// 	log.PanicIf(err)

// 	// Decode from binary.

// 	ti := NewTagIndex()

// 	_, originalIndex, err := Collect(im, ti, rawExif)
// 	log.PanicIf(err)

// 	originalThumbnailData, err := originalIndex.RootIfd.NextIfd.Thumbnail()
// 	log.PanicIf(err)

// 	originalTags := originalIndex.RootIfd.DumpTags()

// 	// Encode back to binary.

// 	ibe := NewIfdByteEncoder()

// 	rootIb := NewIfdBuilderFromExistingChain(originalIndex.RootIfd, nil)

// 	// Update a tag,.

// 	exifBt, err := rootIb.FindTagWithName("ExifTag")
// 	log.PanicIf(err)

// 	ucBt, err := exifBt.value.Ib().FindTagWithName("UserComment")
// 	log.PanicIf(err)

// 	uc := TagUnknownType_9298_UserComment{
// 		EncodingType:  TagUnknownType_9298_UserComment_Encoding_ASCII,
// 		EncodingBytes: []byte("TEST COMMENT"),
// 	}

// 	err = ucBt.SetValue(rootIb.byteOrder, uc)
// 	log.PanicIf(err)

// 	// Encode.

// 	updatedExif, err := ibe.EncodeToExif(rootIb)
// 	log.PanicIf(err)

// 	// Parse again.

// 	_, recoveredIndex, err := Collect(im, ti, updatedExif)
// 	log.PanicIf(err)

// 	recoveredTags := recoveredIndex.RootIfd.DumpTags()

// 	recoveredThumbnailData, err := recoveredIndex.RootIfd.NextIfd.Thumbnail()
// 	log.PanicIf(err)

// 	// Check the thumbnail.

// 	if bytes.Compare(recoveredThumbnailData, originalThumbnailData) != 0 {
// 		t.Fatalf("recovered thumbnail does not match original")
// 	}

// 	// Validate that all of the same IFDs were presented.

// 	originalIfdTags := make([][2]interface{}, 0)
// 	for _, ite := range originalTags {
// 		if ite.ChildIfdPath != "" {
// 			originalIfdTags = append(originalIfdTags, [2]interface{}{ite.IfdPath, ite.TagId})
// 		}
// 	}

// 	recoveredIfdTags := make([][2]interface{}, 0)
// 	for _, ite := range recoveredTags {
// 		if ite.ChildIfdPath != "" {
// 			recoveredIfdTags = append(recoveredIfdTags, [2]interface{}{ite.IfdPath, ite.TagId})
// 		}
// 	}

// 	if reflect.DeepEqual(recoveredIfdTags, originalIfdTags) != true {
// 		fmt.Printf("Original IFD tags:\n\n")

// 		for i, x := range originalIfdTags {
// 			fmt.Printf("  %02d %v\n", i, x)
// 		}

// 		fmt.Printf("\nRecovered IFD tags:\n\n")

// 		for i, x := range recoveredIfdTags {
// 			fmt.Printf("  %02d %v\n", i, x)
// 		}

// 		fmt.Printf("\n")

// 		t.Fatalf("Recovered IFD tags are not correct.")
// 	}

// 	// Validate that all of the tags owned by the IFDs were presented. Note
// 	// that the thumbnail tags are not kept but only produced on the fly, which
// 	// is why we check it above.

// 	if len(recoveredTags) != len(originalTags) {
// 		t.Fatalf("Recovered tag-count does not match original.")
// 	}

// 	for i, recoveredIte := range recoveredTags {
// 		if recoveredIte.ChildIfdPath != "" {
// 			continue
// 		}

// 		originalIte := originalTags[i]

// 		if recoveredIte.IfdPath != originalIte.IfdPath {
// 			t.Fatalf("IfdIdentity not as expected: %s != %s  ITE=%s", recoveredIte.IfdPath, originalIte.IfdPath, recoveredIte)
// 		} else if recoveredIte.TagId != originalIte.TagId {
// 			t.Fatalf("Tag-ID not as expected: %d != %d  ITE=%s", recoveredIte.TagId, originalIte.TagId, recoveredIte)
// 		} else if recoveredIte.TagType != originalIte.TagType {
// 			t.Fatalf("Tag-type not as expected: %d != %d  ITE=%s", recoveredIte.TagType, originalIte.TagType, recoveredIte)
// 		}

// 		originalValueBytes, err := originalIte.ValueBytes(originalIndex.RootIfd.addressableData, originalIndex.RootIfd.ByteOrder)
// 		log.PanicIf(err)

// 		recoveredValueBytes, err := recoveredIte.ValueBytes(recoveredIndex.RootIfd.addressableData, recoveredIndex.RootIfd.ByteOrder)
// 		log.PanicIf(err)

// 		if recoveredIte.TagId == 0x9286 {
// 			expectedValueBytes := make([]byte, 0)

// 			expectedValueBytes = append(expectedValueBytes, []byte{'A', 'S', 'C', 'I', 'I', 0, 0, 0}...)
// 			expectedValueBytes = append(expectedValueBytes, []byte("TEST COMMENT")...)

// 			if bytes.Compare(recoveredValueBytes, expectedValueBytes) != 0 {
// 				t.Fatalf("Recovered UserComment does not have the right value: %v != %v", recoveredValueBytes, expectedValueBytes)
// 			}
// 		} else if bytes.Compare(recoveredValueBytes, originalValueBytes) != 0 {
// 			t.Fatalf("bytes of tag content not correct: %v != %v  ITE=%s", recoveredValueBytes, originalValueBytes, recoveredIte)
// 		}
// 	}
// }

func ExampleIfd_Thumbnail() {
	filepath := path.Join(assetsPath, "NDM_8901.jpg")

	rawExif, err := SearchFileAndExtractExif(filepath)
	log.PanicIf(err)

	im := NewIfdMapping()

	err = LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()

	_, index, err := Collect(im, ti, rawExif)
	log.PanicIf(err)

	thumbnailData, err := index.RootIfd.NextIfd.Thumbnail()
	log.PanicIf(err)

	thumbnailData = thumbnailData
	// Output:
}

func ExampleBuilderTag_SetValue() {
	filepath := path.Join(assetsPath, "NDM_8901.jpg")

	rawExif, err := SearchFileAndExtractExif(filepath)
	log.PanicIf(err)

	im := NewIfdMapping()

	err = LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()

	_, index, err := Collect(im, ti, rawExif)
	log.PanicIf(err)

	// Create builder.

	rootIb := NewIfdBuilderFromExistingChain(index.RootIfd, nil)

	// Find tag to update.

	exifBt, err := rootIb.FindTagWithName("ExifTag")
	log.PanicIf(err)

	ucBt, err := exifBt.value.Ib().FindTagWithName("UserComment")
	log.PanicIf(err)

	// Update the value. Since this is an "undefined"-type tag, we have to use
	// its type-specific struct.

	// TODO(dustin): !! Add an example for setting a non-unknown value, too.
	uc := TagUnknownType_9298_UserComment{
		EncodingType:  TagUnknownType_9298_UserComment_Encoding_ASCII,
		EncodingBytes: []byte("TEST COMMENT"),
	}

	err = ucBt.SetValue(rootIb.byteOrder, uc)
	log.PanicIf(err)

	// Encode.

	ibe := NewIfdByteEncoder()
	updatedExif, err := ibe.EncodeToExif(rootIb)
	log.PanicIf(err)

	updatedExif = updatedExif
	// Output:
}

// ExampleIfdBuilder_SetStandardWithName establishes a chain of `IfdBuilder`
// structs from an existing chain of `Ifd` structs, navigates to the IB
// representing IFD0, updates the ProcessingSoftware tag to a different value,
// encodes down to a new EXIF block, reparses, and validates that the value for
// that tag is what we set it to.
func ExampleIfdBuilder_SetStandardWithName() {
	filepath := path.Join(assetsPath, "NDM_8901.jpg")

	rawExif, err := SearchFileAndExtractExif(filepath)
	log.PanicIf(err)

	// Boilerplate.

	im := NewIfdMapping()

	err = LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()

	// Load current IFDs.

	_, index, err := Collect(im, ti, rawExif)
	log.PanicIf(err)

	ib := NewIfdBuilderFromExistingChain(index.RootIfd, nil)

	// Read the IFD whose tag we want to change.

	// Standard:
	// - "IFD0"
	// - "IFD0/Exif0"
	// - "IFD0/Exif0/Iop0"
	// - "IFD0/GPSInfo0"
	//
	// If the numeric indices are not included, (0) is the default. Note that
	// this isn't strictly necessary in our case since IFD0 is the first IFD anyway, but we're putting it here to show usage.
	ifdPath := "IFD0"

	childIb, err := GetOrCreateIbFromRootIb(ib, ifdPath)
	log.PanicIf(err)

	// There are a few functions that allow you to surgically change the tags in an
	// IFD, but we're just gonna overwrite a tag that has an ASCII value.

	tagName := "ProcessingSoftware"

	err = childIb.SetStandardWithName(tagName, "alternative software")
	log.PanicIf(err)

	// Encode the in-memory representation back down to bytes.

	ibe := NewIfdByteEncoder()

	updatedRawExif, err := ibe.EncodeToExif(ib)
	log.PanicIf(err)

	// Reparse the EXIF to confirm that our value is there.

	_, index, err = Collect(im, ti, updatedRawExif)
	log.PanicIf(err)

	// This isn't strictly necessary for the same reason as above, but it's here
	// for documentation.
	childIfd, err := FindIfdFromRootIfd(index.RootIfd, ifdPath)
	log.PanicIf(err)

	results, err := childIfd.FindTagWithName(tagName)
	log.PanicIf(err)

	for _, ite := range results {
		value, err := childIfd.TagValue(ite)
		log.PanicIf(err)

		stringValue := value.(string)
		fmt.Println(stringValue)
	}

	// Output:
	// alternative software
}

func TestIfdBuilder_CreateIfdBuilderWithExistingIfd(t *testing.T) {
	ti := NewTagIndex()

	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	mi, err := im.GetWithPath(IfdPathStandardGps)
	log.PanicIf(err)

	tagId := mi.TagId

	parentIfd := &Ifd{
		IfdPath:  IfdPathStandard,
		tagIndex: ti,
	}

	ifd := &Ifd{
		IfdPath:   IfdPathStandardGps,
		ByteOrder: TestDefaultByteOrder,
		Offset:    0x123,
		ParentIfd: parentIfd,

		ifdMapping: im,
		tagIndex:   ti,
	}

	ib := NewIfdBuilderWithExistingIfd(ifd)

	if ib.ifdPath != ifd.IfdPath {
		t.Fatalf("IFD-name not correct.")
	} else if ib.ifdTagId != tagId {
		t.Fatalf("IFD tag-ID not correct.")
	} else if ib.byteOrder != ifd.ByteOrder {
		t.Fatalf("IFD byte-order not correct.")
	} else if ib.existingOffset != ifd.Offset {
		t.Fatalf("IFD offset not correct.")
	}
}

func TestNewStandardBuilderTag__OneUnit(t *testing.T) {
	ti := NewTagIndex()

	it, err := ti.Get(IfdPathStandardExif, uint16(0x8833))
	log.PanicIf(err)

	bt := NewStandardBuilderTag(IfdPathStandardExif, it, TestDefaultByteOrder, []uint32{uint32(0x1234)})

	if bt.ifdPath != IfdPathStandardExif {
		t.Fatalf("II in BuilderTag not correct")
	} else if bt.tagId != 0x8833 {
		t.Fatalf("tag-ID not correct")
	} else if bytes.Compare(bt.value.Bytes(), []byte{0x0, 0x0, 0x12, 0x34}) != 0 {
		t.Fatalf("value not correct")
	}
}

func TestNewStandardBuilderTag__TwoUnits(t *testing.T) {
	ti := NewTagIndex()

	it, err := ti.Get(IfdPathStandardExif, uint16(0x8833))
	log.PanicIf(err)

	bt := NewStandardBuilderTag(IfdPathStandardExif, it, TestDefaultByteOrder, []uint32{uint32(0x1234), uint32(0x5678)})

	if bt.ifdPath != IfdPathStandardExif {
		t.Fatalf("II in BuilderTag not correct")
	} else if bt.tagId != 0x8833 {
		t.Fatalf("tag-ID not correct")
	} else if bytes.Compare(bt.value.Bytes(), []byte{
		0x0, 0x0, 0x12, 0x34,
		0x0, 0x0, 0x56, 0x78}) != 0 {
		t.Fatalf("value not correct")
	}
}

func TestIfdBuilder_AddStandardWithName(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	ib := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	err = ib.AddStandardWithName("ProcessingSoftware", "some software")
	log.PanicIf(err)

	if len(ib.tags) != 1 {
		t.Fatalf("Exactly one tag was not found: (%d)", len(ib.tags))
	}

	bt := ib.tags[0]

	if bt.ifdPath != IfdPathStandard {
		t.Fatalf("II not correct: %s", bt.ifdPath)
	} else if bt.tagId != 0x000b {
		t.Fatalf("Tag-ID not correct: (0x%04x)", bt.tagId)
	}

	s := string(bt.value.Bytes())

	if s != "some software\000" {
		t.Fatalf("Value not correct: (%d) [%s]", len(s), s)
	}
}

func TestGetOrCreateIbFromRootIb__Noop(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	rootIb := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	ib, err := GetOrCreateIbFromRootIb(rootIb, "IFD")
	log.PanicIf(err)

	if ib != rootIb {
		t.Fatalf("Expected same IB back from no-op get-or-create.")
	} else if ib.nextIb != nil {
		t.Fatalf("Expected no siblings on IB from no-op get-or-create.")
	} else if len(ib.tags) != 0 {
		t.Fatalf("Expected no new tags on IB from no-op get-or-create.")
	}
}

func TestGetOrCreateIbFromRootIb__FqNoop(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	rootIb := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	ib, err := GetOrCreateIbFromRootIb(rootIb, "IFD0")
	log.PanicIf(err)

	if ib != rootIb {
		t.Fatalf("Expected same IB back from no-op get-or-create.")
	} else if ib.nextIb != nil {
		t.Fatalf("Expected no siblings on IB from no-op get-or-create.")
	} else if len(ib.tags) != 0 {
		t.Fatalf("Expected no new tags on IB from no-op get-or-create.")
	}
}

func TestGetOrCreateIbFromRootIb_InvalidChild(t *testing.T) {
	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	rootIb := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	_, err = GetOrCreateIbFromRootIb(rootIb, "IFD/Invalid")
	if err == nil {
		t.Fatalf("Expected failure for invalid IFD child in IB get-or-create.")
	} else if err.Error() != "ifd child with name [Invalid] not registered: [IFD/Invalid]" {
		log.Panic(err)
	}
}

func TestGetOrCreateIbFromRootIb__Child(t *testing.T) {
	defer func() {
		if state := recover(); state != nil {
			err := log.Wrap(state.(error))
			log.PrintErrorf(err, "Test failure.")
		}
	}()

	im := NewIfdMapping()

	err := LoadStandardIfds(im)
	log.PanicIf(err)

	ti := NewTagIndex()
	rootIb := NewIfdBuilder(im, ti, IfdPathStandard, TestDefaultByteOrder)

	lines := rootIb.DumpToStrings()
	expected := []string{
		"IFD<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-INDEX=(0) IFD-TAG-ID=(0x0000) TAG=[0x0000]>",
	}

	if reflect.DeepEqual(lines, expected) != true {
		fmt.Printf("ACTUAL:\n")
		fmt.Printf("\n")

		for i, line := range lines {
			fmt.Printf("%d: %s\n", i, line)
		}

		fmt.Printf("\n")

		fmt.Printf("EXPECTED:\n")
		fmt.Printf("\n")

		for i, line := range expected {
			fmt.Printf("%d: %s\n", i, line)
		}

		fmt.Printf("\n")

		t.Fatalf("Constructed IFDs not correct.")
	}

	ib, err := GetOrCreateIbFromRootIb(rootIb, "IFD/Exif")
	log.PanicIf(err)

	if ib.fqIfdPath != "IFD/Exif" {
		t.Fatalf("Returned IB does not have the expected path (IFD/Exif).")
	}

	lines = rootIb.DumpToStrings()
	expected = []string{
		"IFD<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-INDEX=(0) IFD-TAG-ID=(0x0000) TAG=[0x0000]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[IFD/Exif] TAG-INDEX=(0) TAG=[0x8769]>",
		"IFD<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-INDEX=(0) IFD-TAG-ID=(0x8769) TAG=[0x8769]>",
	}

	if reflect.DeepEqual(lines, expected) != true {
		fmt.Printf("ACTUAL:\n")
		fmt.Printf("\n")

		for i, line := range lines {
			fmt.Printf("%d: %s\n", i, line)
		}

		fmt.Printf("\n")

		fmt.Printf("EXPECTED:\n")
		fmt.Printf("\n")

		for i, line := range expected {
			fmt.Printf("%d: %s\n", i, line)
		}

		fmt.Printf("\n")

		t.Fatalf("Constructed IFDs not correct.")
	}

	ib, err = GetOrCreateIbFromRootIb(rootIb, "IFD0/Exif/Iop")
	log.PanicIf(err)

	if ib.fqIfdPath != "IFD/Exif/Iop" {
		t.Fatalf("Returned IB does not have the expected path (IFD/Exif/Iop).")
	}

	lines = rootIb.DumpToStrings()
	expected = []string{
		"IFD<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-INDEX=(0) IFD-TAG-ID=(0x0000) TAG=[0x0000]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[IFD/Exif] TAG-INDEX=(0) TAG=[0x8769]>",
		"IFD<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-INDEX=(0) IFD-TAG-ID=(0x8769) TAG=[0x8769]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[IFD/Exif/Iop] TAG-INDEX=(0) TAG=[0xa005]>",
		"IFD<PARENTS=[IFD->IFD/Exif] FQ-IFD-PATH=[IFD/Exif/Iop] IFD-INDEX=(0) IFD-TAG-ID=(0xa005) TAG=[0xa005]>",
	}

	if reflect.DeepEqual(lines, expected) != true {
		fmt.Printf("ACTUAL:\n")
		fmt.Printf("\n")

		for i, line := range lines {
			fmt.Printf("%d: %s\n", i, line)
		}

		fmt.Printf("\n")

		fmt.Printf("EXPECTED:\n")
		fmt.Printf("\n")

		for i, line := range expected {
			fmt.Printf("%d: %s\n", i, line)
		}

		fmt.Printf("\n")

		t.Fatalf("Constructed IFDs not correct.")
	}

	ib, err = GetOrCreateIbFromRootIb(rootIb, "IFD1")
	log.PanicIf(err)

	if ib.fqIfdPath != "IFD1" {
		t.Fatalf("Returned IB does not have the expected path (IFD1).")
	}

	lines = rootIb.DumpToStrings()
	expected = []string{
		"IFD<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-INDEX=(0) IFD-TAG-ID=(0x0000) TAG=[0x0000]>",
		"TAG<PARENTS=[] FQ-IFD-PATH=[IFD] IFD-TAG-ID=(0x0000) CHILD-IFD=[IFD/Exif] TAG-INDEX=(0) TAG=[0x8769]>",
		"IFD<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-INDEX=(0) IFD-TAG-ID=(0x8769) TAG=[0x8769]>",
		"TAG<PARENTS=[IFD] FQ-IFD-PATH=[IFD/Exif] IFD-TAG-ID=(0x8769) CHILD-IFD=[IFD/Exif/Iop] TAG-INDEX=(0) TAG=[0xa005]>",
		"IFD<PARENTS=[IFD->IFD/Exif] FQ-IFD-PATH=[IFD/Exif/Iop] IFD-INDEX=(0) IFD-TAG-ID=(0xa005) TAG=[0xa005]>",
		"IFD<PARENTS=[] FQ-IFD-PATH=[IFD1] IFD-INDEX=(1) IFD-TAG-ID=(0x0000) TAG=[0x0000]>",
	}

	if reflect.DeepEqual(lines, expected) != true {
		fmt.Printf("ACTUAL:\n")
		fmt.Printf("\n")

		for i, line := range lines {
			fmt.Printf("%d: %s\n", i, line)
		}

		fmt.Printf("\n")

		fmt.Printf("EXPECTED:\n")
		fmt.Printf("\n")

		for i, line := range expected {
			fmt.Printf("%d: %s\n", i, line)
		}

		fmt.Printf("\n")

		t.Fatalf("Constructed IFDs not correct.")
	}
}
