/** R157 #513 regression: hash links written by the shell carry the full
 * projectId&versionId selection. With a multi-project catalog the old
 * single-project auto-select could never fire, so every deep link lost its
 * context ("请先在右上角选择版本"). selectionFromQuery must restore the
 * selection from the query and stay fail-closed for unknown ids. */
import test from 'node:test';
import assert from 'node:assert/strict';
import {selectionFromQuery} from './shell/nav-model.ts';

const multiProjectCatalog={
  projects:[{id:'kaicreator-mm/tally'},{id:'mock-owner/project2'}],
  versionsByProject:{
    'kaicreator-mm/tally':[{id:'main'},{id:'version/r157-dev'}],
    'mock-owner/project2':[{id:'main'}]},
  defaultVersionId:'main'};

test('deep link with known project and version restores the full selection',()=>{
  assert.deepEqual(selectionFromQuery(multiProjectCatalog,
    {projectId:'mock-owner/project2',versionId:'main'}),
    {projectId:'mock-owner/project2',versionId:'main'});
  assert.deepEqual(selectionFromQuery(multiProjectCatalog,
    {projectId:'kaicreator-mm/tally',versionId:'version/r157-dev'}),
    {projectId:'kaicreator-mm/tally',versionId:'version/r157-dev'});
});

test('valid project with unknown version keeps the project, version falls back',()=>{
  assert.deepEqual(selectionFromQuery(multiProjectCatalog,
    {projectId:'kaicreator-mm/tally',versionId:'not-a-version'}),
    {projectId:'kaicreator-mm/tally',versionId:null});
});

test('unknown project stays fail-closed (no fake selection)',()=>{
  assert.deepEqual(selectionFromQuery(multiProjectCatalog,
    {projectId:'unknown/repo',versionId:'main'}),
    {projectId:null,versionId:null});
});

test('missing or non-string query values never seed a selection',()=>{
  assert.deepEqual(selectionFromQuery(multiProjectCatalog,{}),
    {projectId:null,versionId:null});
  assert.deepEqual(selectionFromQuery(multiProjectCatalog,
    {projectId:42,versionId:'main'}),
    {projectId:null,versionId:null});
});

test('null catalog (NOT_CONNECTED) never seeds a selection',()=>{
  assert.deepEqual(selectionFromQuery(null,
    {projectId:'kaicreator-mm/tally',versionId:'main'}),
    {projectId:null,versionId:null});
});

test('single-project auto-select path is unchanged (no query needed)',()=>{
  const single={projects:[{id:'only/repo'}],
    versionsByProject:{'only/repo':[{id:'main'}]},defaultVersionId:'main'};
  assert.deepEqual(selectionFromQuery(single,{}),
    {projectId:null,versionId:null});
  assert.deepEqual(selectionFromQuery(single,
    {projectId:'only/repo',versionId:'main'}),
    {projectId:'only/repo',versionId:'main'});
});
